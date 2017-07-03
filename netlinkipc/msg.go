package netlinkipc

import (
	"fmt"
	"time"
)

type CreateMsg struct {
	socketId uint32
	startSeq uint32
	congAlg  string
}

func (c *CreateMsg) New(sid uint32, startSeq uint32, alg string) {
	c.socketId = sid
	c.startSeq = startSeq
	c.congAlg = alg
}

func (c *CreateMsg) SocketId() uint32 {
	return c.socketId
}

func (c *CreateMsg) StartSeq() uint32 {
	return c.startSeq
}

func (c *CreateMsg) CongAlg() string {
	return c.congAlg
}

func (c *CreateMsg) Serialize() ([]byte, error) {
	return msgWriter(nlmsg{
		typ:      CREATE,
		socketId: c.socketId,
		u32s:     []uint32{c.startSeq},
		u64s:     []uint64{},
		str:      "",
	})
}

func (c *CreateMsg) Deserialize(buf []byte) error {
	msg, err := msgReader(buf)
	if err != nil {
		return err
	}

	return c.fromNlmsg(msg)
}

func (c *CreateMsg) fromNlmsg(msg nlmsg) error {
	if msg.typ != CREATE {
		return fmt.Errorf("not a create message: %d", msg.typ)
	}

	c.socketId = msg.socketId
	c.startSeq = msg.u32s[0]
	c.congAlg = msg.str
	return nil
}

type MeasureMsg struct {
	socketId uint32
	ackNo    uint32
	rtt      time.Duration
	rin      uint64
	rout     uint64
}

func (m *MeasureMsg) New(
	sid uint32,
	ack uint32,
	t time.Duration,
	rin uint64,
	rout uint64,
) {
	m.socketId = sid
	m.ackNo = ack
	m.rtt = t
	m.rin = rin
	m.rout = rout
}

func (m *MeasureMsg) SocketId() uint32 {
	return m.socketId
}

func (m *MeasureMsg) AckNo() uint32 {
	return m.ackNo
}

func (m *MeasureMsg) Rtt() time.Duration {
	return m.rtt
}

func (m *MeasureMsg) Rin() uint64 {
	return m.rin
}

func (m *MeasureMsg) Rout() uint64 {
	return m.rout
}

func (m *MeasureMsg) Serialize() ([]byte, error) {
	return msgWriter(nlmsg{
		typ:      MEASURE,
		socketId: m.socketId,
		u32s:     []uint32{m.ackNo, uint32(m.rtt.Nanoseconds())},
		u64s:     []uint64{m.rin, m.rout},
		str:      "",
	})
}

func (m *MeasureMsg) Deserialize(buf []byte) error {
	msg, err := msgReader(buf)
	if err != nil {
		return err
	}

	return m.fromNlmsg(msg)
}

func (m *MeasureMsg) fromNlmsg(nlm nlmsg) error {
	if nlm.typ != MEASURE {
		return fmt.Errorf("not a measure message: %d", nlm.typ)
	}

	m.socketId = nlm.socketId
	m.ackNo = nlm.u32s[0]
	m.rtt = time.Duration(nlm.u32s[1]) * time.Microsecond
	m.rin = nlm.u64s[0]
	m.rout = nlm.u64s[1]
	return nil
}

type CwndMsg struct {
	socketId uint32
	cwnd     uint32
}

func (c *CwndMsg) New(sid uint32, cw uint32) {
	c.socketId = sid
	c.cwnd = cw
}

func (c *CwndMsg) SocketId() uint32 {
	return c.socketId
}

func (c *CwndMsg) Cwnd() uint32 {
	return c.cwnd
}

func (c *CwndMsg) Serialize() ([]byte, error) {
	return msgWriter(nlmsg{
		typ:      CWND,
		socketId: c.socketId,
		u32s:     []uint32{c.cwnd},
		u64s:     []uint64{},
		str:      "",
	})
}

func (c *CwndMsg) Deserialize(buf []byte) error {
	msg, err := msgReader(buf)
	if err != nil {
		return err
	}

	return c.fromNlmsg(msg)
}

func (c *CwndMsg) fromNlmsg(msg nlmsg) error {
	if msg.typ != CWND {
		return fmt.Errorf("not a cwnd message: %d", msg.typ)
	}

	c.socketId = msg.socketId
	c.cwnd = msg.u32s[0]
	return nil
}

type DropMsg struct {
	socketId uint32
	event    string
}

func (d *DropMsg) New(sid uint32, ev string) {
	d.socketId = sid
	d.event = ev
}

func (d *DropMsg) SocketId() uint32 {
	return d.socketId
}

func (d *DropMsg) Event() string {
	return d.event
}

func (d *DropMsg) Serialize() ([]byte, error) {
	return msgWriter(nlmsg{
		typ:      DROP,
		socketId: d.socketId,
		u32s:     []uint32{},
		u64s:     []uint64{},
		str:      d.event,
	})
}

func (d *DropMsg) Deserialize(buf []byte) error {
	msg, err := msgReader(buf)
	if err != nil {
		return err
	}

	return d.fromNlmsg(msg)
}

func (d *DropMsg) fromNlmsg(msg nlmsg) error {
	if msg.typ != DROP {
		return fmt.Errorf("not a drop message: %d", msg.typ)
	}

	d.socketId = msg.socketId
	d.event = msg.str
	return nil
}
