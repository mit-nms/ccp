package unixsocket

import (
	"fmt"
	"time"

	capnpMsg "ccp/capnpMsg"

	"zombiezen.com/go/capnproto2"
)

func makeCreateMsg(socketId uint32, startSeq uint32, alg string) (*capnp.Message, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	createMsg, err := capnpMsg.NewRootUIntStrMsg(seg)
	if err != nil {
		return nil, err
	}

	createMsg.SetSocketId(socketId)
	createMsg.SetNumUInt32(startSeq)
	createMsg.SetVal(alg)
	createMsg.SetType(capnpMsg.MsgType_create)

	return msg, nil
}

func (c *CreateMsg) readCreateMsg(msg *capnp.Message) (err error) {
	defer func() {
		if panic := recover(); panic != nil {
			err = panic.(error)
		}
	}()

	createMsg, err := capnpMsg.ReadRootUIntStrMsg(msg)
	if err != nil {
		return err
	}

	if createMsg.Type() != capnpMsg.MsgType_create {
		return fmt.Errorf("Message not of type Create: %v", createMsg.Type())
	}

	stSq := createMsg.NumUInt32()
	alg, err := createMsg.Val()
	if err != nil {
		return err
	}

	c.socketId = createMsg.SocketId()
	c.startSeq = stSq
	c.congAlg = alg
	return nil
}

func makeNotifyMeasureMsg(socketId uint32, ackNo uint32, rtt time.Duration) (*capnp.Message, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	notifyAckMsg, err := capnpMsg.NewRootUInt32UInt64Msg(seg)
	if err != nil {
		return nil, err
	}

	notifyAckMsg.SetType(capnpMsg.MsgType_ack)
	notifyAckMsg.SetSocketId(socketId)
	notifyAckMsg.SetNumInt32(ackNo)
	notifyAckMsg.SetNumInt64(uint64(rtt.Nanoseconds()))

	return msg, nil
}

func (a *MeasureMsg) readMeasureMsg(msg *capnp.Message) (err error) {
	defer func() {
		if panic := recover(); panic != nil {
			err = panic.(error)
		}
	}()

	ackMsg, err := capnpMsg.ReadRootUInt32UInt64Msg(msg)
	if err != nil {
		return err
	}

	if ackMsg.Type() != capnpMsg.MsgType_ack {
		return fmt.Errorf("Message not of type Ack: %v", ackMsg.Type())
	}

	a.socketId = ackMsg.SocketId()
	a.ackNo = ackMsg.NumInt32()
	a.rtt = time.Duration(ackMsg.NumInt64())
	return nil
}

func makeCwndMsg(socketId uint32, cwnd uint32) (*capnp.Message, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	cwndMsg, err := capnpMsg.NewRootUIntMsg(seg)
	if err != nil {
		return nil, err
	}

	cwndMsg.SetType(capnpMsg.MsgType_cwnd)
	cwndMsg.SetSocketId(socketId)
	cwndMsg.SetVal(cwnd)

	return msg, nil
}

func (c *CwndMsg) readCwndMsg(msg *capnp.Message) (err error) {
	defer func() {
		if panic := recover(); panic != nil {
			err = panic.(error)
		}
	}()

	cwndUpdateMsg, err := capnpMsg.ReadRootUIntMsg(msg)
	if err != nil {
		return err
	}

	if cwndUpdateMsg.Type() != capnpMsg.MsgType_cwnd {
		return fmt.Errorf("Message not of type Cwnd: %v", cwndUpdateMsg.Type())
	}

	c.socketId = cwndUpdateMsg.SocketId()
	c.cwnd = cwndUpdateMsg.Val()
	return nil
}

func makeDropMsg(socketId uint32, ev string) (*capnp.Message, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	dropMsg, err := capnpMsg.NewRootStrMsg(seg)
	if err != nil {
		return nil, err
	}

	dropMsg.SetSocketId(socketId)
	dropMsg.SetVal(ev)
	dropMsg.SetType(capnpMsg.MsgType_drop)

	return msg, nil
}

func (d *DropMsg) readDropMsg(msg *capnp.Message) (err error) {
	defer func() {
		if panic := recover(); panic != nil {
			err = panic.(error)
		}
	}()

	dropMsg, err := capnpMsg.ReadRootStrMsg(msg)
	if err != nil {
		return err
	}

	if dropMsg.Type() != capnpMsg.MsgType_drop {
		return fmt.Errorf("Message not of type Drop: %v", dropMsg.Type())
	}

	ev, err := dropMsg.Val()
	if err != nil {
		return err
	}

	d.socketId = dropMsg.SocketId()
	d.event = ev
	return nil
}
