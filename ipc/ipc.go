package ipc

import (
	"fmt"

	capnpMsg "ccp/capnpMsg"

	"zombiezen.com/go/capnproto2"
)

type AckMsg struct {
	SocketId uint32
	AckNo    uint32
}

type CwndMsg struct {
	SocketId uint32
	Cwnd     uint32
}

type IpcLayer interface {
	SendAckMsg(socketId uint32, ackNo uint32) error
	ListenAckMsg() (chan AckMsg, error)
	SendCwndMsg(socketId uint32, cwnd uint32) error
	ListenCwndMsg() (chan CwndMsg, error)
	Close() error
}

type BaseIpcLayer struct {
	AckNotify  chan AckMsg
	CwndNotify chan CwndMsg
}

func (b *BaseIpcLayer) SendAckMsg(socketId uint32, ackNo uint32) error {
	return fmt.Errorf("SendAckMsg unimplemented in BaseIpcLayer")
}

func (b *BaseIpcLayer) SendCwndMsg(socketId uint32, ackNo uint32) error {
	return fmt.Errorf("SendCwndMsg unimplemented in BaseIpcLayer")
}

func (b *BaseIpcLayer) ListenAckMsg() (chan AckMsg, error) {
	return b.AckNotify, nil
}

func (b *BaseIpcLayer) ListenCwndMsg() (chan CwndMsg, error) {
	return b.CwndNotify, nil
}

func (b *BaseIpcLayer) Close() error {
	return fmt.Errorf("Close unimplemented in BaseIpcLayer")
}

func (b *BaseIpcLayer) Parse(msg *capnp.Message) error {
	if sid, cwnd, err := ReadCwndMsg(msg); err == nil {
		msg := CwndMsg{
			SocketId: sid,
			Cwnd:     cwnd,
		}

		select {
		case b.CwndNotify <- msg:
		default:
		}

		return nil
	}

	if sid, ack, err := ReadAckMsg(msg); err == nil {
		msg := AckMsg{
			SocketId: sid,
			AckNo:    ack,
		}

		select {
		case b.AckNotify <- msg:
		default:
		}

		return nil
	} else {
		return err
	}
}

func MakeNotifyAckMsg(socketId uint32, ackNo uint32) (*capnp.Message, error) {
	// Cap'n Proto! Message passing interface to mmap file
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	notifyAckMsg, err := capnpMsg.NewRootNotifyAckMsg(seg)
	if err != nil {
		return nil, err
	}

	notifyAckMsg.SetSocketId(socketId)
	notifyAckMsg.SetAckNo(ackNo)

	return msg, nil
}

func ReadAckMsg(msg *capnp.Message) (uint32, uint32, error) {
	ackMsg, err := capnpMsg.ReadRootNotifyAckMsg(msg)
	if err != nil {
		return 0, 0, err
	}

	return ackMsg.SocketId(), ackMsg.AckNo(), nil
}

func MakeCwndMsg(socketId uint32, cwnd uint32) (*capnp.Message, error) {
	// Cap'n Proto! Message passing interface to mmap file
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	cwndMsg, err := capnpMsg.NewRootSetCwndMsg(seg)
	if err != nil {
		return nil, err
	}

	cwndMsg.SetSocketId(socketId)
	cwndMsg.SetCwnd(cwnd)

	return msg, nil
}

func ReadCwndMsg(msg *capnp.Message) (uint32, uint32, error) {
	cwndUpdateMsg, err := capnpMsg.ReadRootSetCwndMsg(msg)
	if err != nil {
		return 0, 0, err
	}

	return cwndUpdateMsg.SocketId(), cwndUpdateMsg.Cwnd(), nil
}
