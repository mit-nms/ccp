package netlinkipc

import (
	"fmt"
	"time"
)

// Message serialized format
// -----------------------------------------
// | Msg Type | Len (B)  | Msg Specific...
// | (1 B)    | (1 B)    |
// -----------------------------------------

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
	return writeUInt32AndUInt32AndString(
		CREATE,
		c.socketId,
		c.startSeq,
		c.congAlg,
	)
}

func (c *CreateMsg) Deserialize(buf []byte) error {
	t, sid, stSq, alg := readUInt32AndUInt32AndString(buf)
	if t != CREATE {
		return fmt.Errorf("not a create message: %d", t)
	}

	c.socketId = sid
	c.startSeq = stSq
	c.congAlg = alg
	return nil
}

type AckMsg struct {
	socketId uint32
	ackNo    uint32
	rtt      time.Duration
}

func (a *AckMsg) New(sid uint32, ack uint32, t time.Duration) {
	a.socketId = sid
	a.ackNo = ack
	a.rtt = t
}

func (a *AckMsg) SocketId() uint32 {
	return a.socketId
}

func (a *AckMsg) AckNo() uint32 {
	return a.ackNo
}

func (a *AckMsg) Rtt() time.Duration {
	return a.rtt
}

func (a *AckMsg) Serialize() ([]byte, error) {
	return writeUInt32AndUInt32AndUInt64(
		ACK,
		a.socketId,
		a.ackNo,
		uint64(a.rtt.Nanoseconds()),
	)
}

func (a *AckMsg) Deserialize(buf []byte) error {
	t, sid, ack, tm := readUInt32AndUInt32AndUInt64(buf)
	if t != ACK {
		return fmt.Errorf("not a ack message: %d", t)
	}

	a.socketId = sid
	a.ackNo = ack
	a.rtt = time.Duration(tm) * time.Nanosecond

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
	return writeUInt32AndUInt32(
		CWND,
		c.socketId,
		c.cwnd,
	)
}

func (c *CwndMsg) Deserialize(buf []byte) error {
	t, sid, cw := readUInt32AndUInt32(buf)
	if t != CWND {
		return fmt.Errorf("not a cwnd message: %d", t)
	}

	c.socketId = sid
	c.cwnd = cw

	return nil
}

type DropMsg struct {
	socketId uint32
	event    string
}

func (c *DropMsg) New(sid uint32, ev string) {
	c.socketId = sid
	c.event = ev
}

func (c *DropMsg) SocketId() uint32 {
	return c.socketId
}

func (c *DropMsg) Event() string {
	return c.event
}

func (c *DropMsg) Serialize() ([]byte, error) {
	return writeUInt32AndString(
		DROP,
		c.socketId,
		c.event,
	)
}

func (c *DropMsg) Deserialize(buf []byte) error {
	t, sid, ev := readUInt32AndString(buf)
	if t != DROP {
		return fmt.Errorf("not a create message: %d", t)
	}

	c.socketId = sid
	c.event = ev
	return nil
}
