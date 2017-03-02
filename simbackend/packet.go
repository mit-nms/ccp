package simbackend

import (
	"bytes"
	"encoding/binary"

	"github.mit.edu/hari/nimbus-cc/packetops"
)

// 1463 = 1500 - 28 (ip + udp) - 10 (my header)
const PACKET_SIZE = 1462

type PacketFlag uint8

const (
	SYN PacketFlag = iota
	SYNACK
	ACK
	FIN
)

// Implement packetops.Packet
type Packet struct {
	SeqNo   uint32     // 32 bits = 4 bytes
	AckNo   uint32     // 32 bits = 4 bytes
	Flag    PacketFlag // Upper 4 bits of Length int below = 1 byte
	Length  uint16     // Only use bottom 12 bits! Max size = 2^12 = 4096. 12 bits = 1.5 bytes
	Payload []byte
}

func (pkt *Packet) Encode(
	size int,
) (*packetops.RawPacket, error) {
	b, err := encode(*pkt)
	return &packetops.RawPacket{Buf: b}, err
}

func encode(p Packet) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, p.SeqNo)
	if err != nil {
		return buf.Bytes(), err
	}

	err = binary.Write(buf, binary.LittleEndian, p.AckNo)
	if err != nil {
		return buf.Bytes(), err
	}

	// ensure only bottom 12 bits used
	p.Length = p.Length & 0x0fff
	// ensure only bottom 4 bits used
	flag := (uint16(p.Flag) & 0x3) << 12

	field := flag | p.Length // uint16, top 4 bits flag, bottom 12 bits len

	err = binary.Write(buf, binary.LittleEndian, field)
	if err != nil {
		return buf.Bytes(), err
	}

	buf.Write(p.Payload)

	return buf.Bytes(), err
}

func (pkt *Packet) Decode(
	r *packetops.RawPacket,
) error {
	p, err := decode(r.Buf)
	if err != nil {
		return err
	}

	pkt.SeqNo = p.SeqNo
	pkt.AckNo = p.AckNo
	pkt.Flag = p.Flag
	pkt.Length = p.Length
	pkt.Payload = p.Payload

	return nil
}

func decode(b []byte) (Packet, error) {
	var p Packet
	buf := bytes.NewReader(b)

	err := binary.Read(buf, binary.LittleEndian, &p.SeqNo)
	if err != nil {
		return p, err
	}

	err = binary.Read(buf, binary.LittleEndian, &p.AckNo)
	if err != nil {
		return p, err
	}

	var field uint16
	err = binary.Read(buf, binary.LittleEndian, &field)
	if err != nil {
		return p, err
	}

	p.Length = field & 0xfff
	p.Flag = PacketFlag((field & 0xf000) >> 12)

	// SeqNo + AckNo + (Flag,Length) = 10 bytes
	p.Payload = make([]byte, len(b)-10)
	copy(p.Payload, b[10:])

	return p, err
}
