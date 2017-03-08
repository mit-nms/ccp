package ipc

import (
	"fmt"

	capnpMsg "ccp/capnpMsg"

	log "github.com/Sirupsen/logrus"
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
	SendCwndMsg(socketId uint32, cwnd uint32) error
	ListenAckMsg() (chan AckMsg, error)
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
	if msg, err := ReadCwndMsg(msg); err == nil {
		select {
		case b.CwndNotify <- msg:
		default:
			log.WithFields(log.Fields{
				"msg": msg,
			}).Warn("dropping message")
		}

		return nil
	}

	if msg, err := ReadAckMsg(msg); err == nil {
		select {
		case b.AckNotify <- msg:
		default:
			log.WithFields(log.Fields{
				"msg": msg,
			}).Warn("dropping message")
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

	notifyAckMsg, err := capnpMsg.NewRootUIntMsg(seg)
	if err != nil {
		return nil, err
	}

	notifyAckMsg.SetType(capnpMsg.UIntMsgType_ack)
	notifyAckMsg.SetSocketId(socketId)
	notifyAckMsg.SetVal(ackNo)

	return msg, nil
}

func MakeCwndMsg(socketId uint32, cwnd uint32) (*capnp.Message, error) {
	// Cap'n Proto! Message passing interface to mmap file
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	cwndMsg, err := capnpMsg.NewRootUIntMsg(seg)
	if err != nil {
		return nil, err
	}

	cwndMsg.SetType(capnpMsg.UIntMsgType_cwnd)
	cwndMsg.SetSocketId(socketId)
	cwndMsg.SetVal(cwnd)

	return msg, nil
}

func ReadAckMsg(msg *capnp.Message) (AckMsg, error) {
	ackMsg, err := capnpMsg.ReadRootUIntMsg(msg)
	if err != nil {
		return AckMsg{}, err
	}

	if ackMsg.Type() != capnpMsg.UIntMsgType_ack {
		return AckMsg{}, fmt.Errorf("Message not of type Ack: %v", ackMsg.Type())
	}

	return AckMsg{SocketId: ackMsg.SocketId(), AckNo: ackMsg.Val()}, nil
}

func ReadCwndMsg(msg *capnp.Message) (CwndMsg, error) {
	cwndUpdateMsg, err := capnpMsg.ReadRootUIntMsg(msg)
	if err != nil {
		return CwndMsg{}, err
	}

	if cwndUpdateMsg.Type() != capnpMsg.UIntMsgType_cwnd {
		return CwndMsg{}, fmt.Errorf("Message not of type Cwnd: %v", cwndUpdateMsg.Type())
	}

	return CwndMsg{SocketId: cwndUpdateMsg.SocketId(), Cwnd: cwndUpdateMsg.Val()}, nil
}
