package unixsocket

import (
	"time"

	"zombiezen.com/go/capnproto2"
)

type CreateMsg struct {
	socketId uint32
	congAlg  string
}

func (c *CreateMsg) New(sid uint32, alg string) {
	c.socketId = sid
	c.congAlg = alg
}

func (c *CreateMsg) SocketId() uint32 {
	return c.socketId
}

func (c *CreateMsg) CongAlg() string {
	return c.congAlg
}

func (c *CreateMsg) Serialize() ([]byte, error) {
	cmsg, err := makeCreateMsg(c.socketId, c.congAlg)
	if err != nil {
		return []byte{}, err
	}

	buf, err := cmsg.Marshal()
	if err != nil {
		return []byte{}, err
	}

	return buf, nil
}

func (c *CreateMsg) Deserialize(buf []byte) error {
	msg, err := capnp.Unmarshal(buf)
	if err != nil {
		return err
	}

	err = c.readCreateMsg(msg)
	if err != nil {
		return err
	}

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
	cmsg, err := makeNotifyAckMsg(a.socketId, a.ackNo, a.rtt)
	if err != nil {
		return []byte{}, err
	}

	buf, err := cmsg.Marshal()
	if err != nil {
		return []byte{}, err
	}

	return buf, nil
}

func (a *AckMsg) Deserialize(buf []byte) error {
	msg, err := capnp.Unmarshal(buf)
	if err != nil {
		return err
	}

	err = a.readAckMsg(msg)
	if err != nil {
		return err
	}

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
	cmsg, err := makeCwndMsg(c.socketId, c.cwnd)
	if err != nil {
		return []byte{}, err
	}

	buf, err := cmsg.Marshal()
	if err != nil {
		return []byte{}, err
	}

	return buf, nil
}

func (c *CwndMsg) Deserialize(buf []byte) error {
	msg, err := capnp.Unmarshal(buf)
	if err != nil {
		return err
	}

	err = c.readCwndMsg(msg)
	if err != nil {
		return err
	}

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
	cmsg, err := makeDropMsg(c.socketId, c.event)
	if err != nil {
		return []byte{}, err
	}

	buf, err := cmsg.Marshal()
	if err != nil {
		return []byte{}, err
	}

	return buf, nil
}

func (c *DropMsg) Deserialize(buf []byte) error {
	msg, err := capnp.Unmarshal(buf)
	if err != nil {
		return err
	}

	err = c.readDropMsg(msg)
	if err != nil {
		return err
	}

	return nil
}
