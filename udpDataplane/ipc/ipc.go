package ipc

import (
	capnpMsg "udpDataplane/capnpMsg"

	"zombiezen.com/go/capnproto2"
)

type IpcLayer interface {
	Send(socketId uint32, ackNo uint32) error
	Listen() (chan uint32, error)
	Close() error
}

func NotifyAckMsg(socketId uint32, ackNo uint32) (*capnp.Message, error) {
	// Cap'n Proto! Message passing interface to mmap file
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	notifyAckMsg, err := capnpMsg.NewRootNotifyAckMsg(seg)
	if err != nil {
		return nil, err
	}

	notifyAckMsg.SetSocketId(4)
	notifyAckMsg.SetAckNo(ackNo)

	return msg, nil
}

func ReadCwndMsg(msg *capnp.Message) (uint32, error) {
	cwndUpdateMsg, err := capnpMsg.ReadRootSetCwndMsg(msg)
	if err != nil {
		return 0, err
	}

	return cwndUpdateMsg.Cwnd(), nil
}
