package ipc

import (
	"fmt"

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
}

type CwndMsg struct {
	SocketId uint32
	Cwnd     uint32
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

func makeNotifyAckMsg(socketId uint32, ackNo uint32) (*capnp.Message, error) {
	// Cap'n Proto! Message passing interface to mmap file
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	notifyAckMsg, err := capnpMsg.NewRootUIntMsg(seg)
	if err != nil {
		return nil, err
	}

	notifyAckMsg.SetType(capnpMsg.MsgType_ack)
	notifyAckMsg.SetSocketId(socketId)
	notifyAckMsg.SetVal(ackNo)

	return msg, nil
}

func readAckMsg(msg *capnp.Message) (a AckMsg, e error) {
	defer func() {
		if panic := recover(); panic != nil {
			e = panic.(error)
		}
	}()

	ackMsg, err := capnpMsg.ReadRootUIntMsg(msg)
	if err != nil {
		return AckMsg{}, err
	}

	if ackMsg.Type() != capnpMsg.MsgType_ack {
		return AckMsg{}, fmt.Errorf("Message not of type Ack: %v", ackMsg.Type())
	}

	return AckMsg{SocketId: ackMsg.SocketId(), AckNo: ackMsg.Val()}, nil
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
