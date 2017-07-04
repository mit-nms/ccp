package ipc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// the internal serialization logic

type msgType uint8

const (
	CREATE msgType = iota
	MEASURE
	DROP
	SET
)

/* Messages: header followed by 0+ uint32s, then 0+ uint64s, then 0-1 strings
 */
type ipcMsg struct {
	typ      msgType
	len      uint8
	socketId uint32
	u32s     []uint32
	u64s     []uint64
	str      string
}

/* (type, len, socket_id) header
 * -----------------------------------
 * | Msg Type | Len (B)  | Uint32    |
 * | (1 B)    | (1 B)    | (32 bits) |
 * -----------------------------------
 * total: 6 Bytes
 */
func readHeader(b []byte) (
	typ msgType,
	l uint8,
	socketId uint32,
	err error,
) {
	err = nil
	if len(b) < 6 {
		err = fmt.Errorf("unable to read header")
		return
	}

	hdr := bytes.NewBuffer(b[:6])
	err = binary.Read(hdr, binary.LittleEndian, &typ)
	err = binary.Read(hdr, binary.LittleEndian, &l)
	err = binary.Read(hdr, binary.LittleEndian, &socketId)
	return
}

func writeHeader(
	typ msgType,
	len uint8,
	socketId uint32,
) (b []byte) {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint8(typ))
	binary.Write(buf, binary.LittleEndian, len)
	binary.Write(buf, binary.LittleEndian, socketId)

	return buf.Bytes()
}

func msgReader(buf []byte) (msg ipcMsg, err error) {
	typ, l, socketId, err := readHeader(buf)
	if err != nil {
		return ipcMsg{}, err
	}

	msg = ipcMsg{
		typ:      typ,
		len:      l,
		socketId: socketId,
		u32s:     make([]uint32, 0),
		u64s:     make([]uint64, 0),
		str:      "",
	}

	var numU32, numU64 int
	var hasStr bool
	switch typ {
	case CREATE:
		numU32 = 1
		numU64 = 0
		hasStr = true
	case DROP:
		numU32 = 0
		numU64 = 0
		hasStr = true
	case MEASURE:
		numU32 = 2
		numU64 = 2
		hasStr = false
	case SET:
		numU32 = 1
		numU64 = 0
		hasStr = true
	default:
		return ipcMsg{}, fmt.Errorf("malformed message")
	}

	payload := bytes.NewBuffer(buf[6:])
	for i := 0; i < numU32; i++ {
		var u uint32
		binary.Read(payload, binary.LittleEndian, &u)
		msg.u32s = append(msg.u32s, u)
	}

	for i := 0; i < numU64; i++ {
		var u uint64
		binary.Read(payload, binary.LittleEndian, &u)
		msg.u64s = append(msg.u64s, u)
	}

	if hasStr {
		s := make([]byte, int(msg.len)-6-numU32*4-numU64*8)
		binary.Read(payload, binary.LittleEndian, &s)

		// remove null terminator
		if s[len(s)-1] == byte(0) {
			s = s[:len(s)-1]
		}
		msg.str = string(s)
	}

	return
}

func (i *Ipc) demux(ch chan []byte) {
	for buf := range ch {
		ipcm, err := msgReader(buf)
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
				"buf": buf,
			}).Warn("failed to parse message")
			continue
		}
		switch ipcm.typ {
		case MEASURE:
			i.MeasureNotify <- MeasureMsg{
				socketId: ipcm.socketId,
				ackNo:    ipcm.u32s[0],
				rtt:      time.Duration(ipcm.u32s[1]) * time.Nanosecond,
				rin:      ipcm.u64s[0],
				rout:     ipcm.u64s[1],
			}
		case DROP:
			i.DropNotify <- DropMsg{
				socketId: ipcm.socketId,
				event:    ipcm.str,
			}
		case CREATE:
			i.CreateNotify <- CreateMsg{
				socketId: ipcm.socketId,
				startSeq: ipcm.u32s[0],
				congAlg:  ipcm.str,
			}
		case SET:
			i.SetNotify <- SetMsg{
				socketId:   ipcm.socketId,
				cwndOrRate: ipcm.u32s[0],
				mode:       ipcm.str,
			}
		}
	}
}

func msgWriter(msg ipcMsg) ([]byte, error) {
	// header: 6 Bytes
	switch {
	case msg.typ == CREATE && len(msg.u32s) == 1 && len(msg.u64s) == 0 && msg.str != "":
		// + 1 uint32, + string
		msg.len = 10 + uint8(len(msg.str))
	case msg.typ == DROP && len(msg.u32s) == 0 && len(msg.u64s) == 0 && msg.str != "":
		// + string
		msg.len = uint8(6 + len(msg.str))
	case msg.typ == MEASURE && len(msg.u32s) == 2 && len(msg.u64s) == 2 && msg.str == "":
		// + 2 uint32, + 2 uint64, no string
		// 6 + 8 + 16 = 30
		msg.len = 30
	case msg.typ == SET && len(msg.u32s) == 1 && len(msg.u64s) == 0 && msg.str != "":
		// + 1 uint32, + string
		msg.len = 10 + uint8(len(msg.str))
	default:
		return nil, fmt.Errorf("Invalid message")
	}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint8(msg.typ))
	binary.Write(buf, binary.LittleEndian, msg.len)
	binary.Write(buf, binary.LittleEndian, msg.socketId)

	for _, val := range msg.u32s {
		binary.Write(buf, binary.LittleEndian, val)
	}

	for _, val := range msg.u64s {
		binary.Write(buf, binary.LittleEndian, val)
	}

	if msg.str != "" {
		binary.Write(buf, binary.LittleEndian, []byte(msg.str))
	}

	return buf.Bytes(), nil
}
