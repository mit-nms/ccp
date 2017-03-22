package ipc

import (
	"fmt"
	"time"

	capnpMsg "ccp/capnpMsg"

	"zombiezen.com/go/capnproto2"
)

type CreateMsg struct {
	SocketId uint32
	CongAlg  string
}

type AckMsg struct {
	SocketId uint32
	AckNo    uint32
	Rtt      time.Duration
}

type CwndMsg struct {
	SocketId uint32
	Cwnd     uint32
}

type DropMsg struct {
	SocketId uint32
	Event    string
}

func makeCreateMsg(socketId uint32, alg string) (*capnp.Message, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	createMsg, err := capnpMsg.NewRootStrMsg(seg)
	if err != nil {
		return nil, err
	}

	createMsg.SetSocketId(socketId)
	createMsg.SetVal(alg)
	createMsg.SetType(capnpMsg.MsgType_create)

	return msg, nil
}

func readCreateMsg(msg *capnp.Message) (c CreateMsg, e error) {
	defer func() {
		if panic := recover(); panic != nil {
			e = panic.(error)
		}
	}()

	createMsg, err := capnpMsg.ReadRootStrMsg(msg)
	if err != nil {
		return CreateMsg{}, err
	}

	if createMsg.Type() != capnpMsg.MsgType_create {
		return CreateMsg{}, fmt.Errorf("Message not of type Create: %v", createMsg.Type())
	}

	alg, err := createMsg.Val()
	if err != nil {
		return CreateMsg{}, err
	}

	return CreateMsg{SocketId: createMsg.SocketId(), CongAlg: alg}, nil
}

func makeNotifyAckMsg(socketId uint32, ackNo uint32, rtt time.Duration) (*capnp.Message, error) {
	// Cap'n Proto! Message passing interface to mmap file
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

func readAckMsg(msg *capnp.Message) (a AckMsg, e error) {
	defer func() {
		if panic := recover(); panic != nil {
			e = panic.(error)
		}
	}()

	ackMsg, err := capnpMsg.ReadRootUInt32UInt64Msg(msg)
	if err != nil {
		return AckMsg{}, err
	}

	if ackMsg.Type() != capnpMsg.MsgType_ack {
		return AckMsg{}, fmt.Errorf("Message not of type Ack: %v", ackMsg.Type())
	}

	return AckMsg{
		SocketId: ackMsg.SocketId(),
		AckNo:    ackMsg.NumInt32(),
		Rtt:      time.Duration(ackMsg.NumInt64()),
	}, nil
}

func makeCwndMsg(socketId uint32, cwnd uint32) (*capnp.Message, error) {
	// Cap'n Proto! Message passing interface to mmap file
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

func readCwndMsg(msg *capnp.Message) (c CwndMsg, e error) {
	defer func() {
		if panic := recover(); panic != nil {
			e = panic.(error)
		}
	}()

	cwndUpdateMsg, err := capnpMsg.ReadRootUIntMsg(msg)
	if err != nil {
		return CwndMsg{}, err
	}

	if cwndUpdateMsg.Type() != capnpMsg.MsgType_cwnd {
		return CwndMsg{}, fmt.Errorf("Message not of type Cwnd: %v", cwndUpdateMsg.Type())
	}

	return CwndMsg{SocketId: cwndUpdateMsg.SocketId(), Cwnd: cwndUpdateMsg.Val()}, nil
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

func readDropMsg(msg *capnp.Message) (d DropMsg, e error) {
	defer func() {
		if panic := recover(); panic != nil {
			e = panic.(error)
		}
	}()

	dropMsg, err := capnpMsg.ReadRootStrMsg(msg)
	if err != nil {
		return DropMsg{}, err
	}

	if dropMsg.Type() != capnpMsg.MsgType_drop {
		return DropMsg{}, fmt.Errorf("Message not of type Drop: %v", dropMsg.Type())
	}

	ev, err := dropMsg.Val()
	if err != nil {
		return DropMsg{}, err
	}

	return DropMsg{SocketId: dropMsg.SocketId(), Event: ev}, nil
}
