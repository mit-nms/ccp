package ipc

import (
	"time"

	"ccp/ipcBackend"
	"ccp/unixsocket"

	"zombiezen.com/go/capnproto2"
)

type Ipc struct {
	CreateNotify chan CreateMsg
	AckNotify    chan AckMsg
	CwndNotify   chan CwndMsg
	DropNotify   chan DropMsg

	backend ipcbackend.Backend
}

// handle of IPC to pass to CC implementations
type SendOnly interface {
	SendCwndMsg(socketId uint32, cwnd uint32) error
}

func SetupCcpListen() (*Ipc, error) {
	back, err := unixsocket.New().SetupListen("ccp-in", 0).SetupFinish()
	if err != nil {
		return nil, err
	}

	return SetupWithBackend(back)
}

func SetupCcpSend(sockid uint32) (SendOnly, error) {
	back, err := unixsocket.New().SetupSend("ccp-out", sockid).SetupFinish()
	if err != nil {
		return nil, err
	}

	return SetupWithBackend(back)
}

func SetupCli(sockid uint32) (*Ipc, error) {
	back, err := unixsocket.New().SetupSend("ccp-in", 0).SetupListen("ccp-out", sockid).SetupFinish()
	if err != nil {
		return nil, err
	}

	return SetupWithBackend(back)
}

func SetupWithBackend(back ipcbackend.Backend) (*Ipc, error) {
	i := &Ipc{
		CreateNotify: make(chan CreateMsg),
		AckNotify:    make(chan AckMsg),
		CwndNotify:   make(chan CwndMsg),
		DropNotify:   make(chan DropMsg),
		backend:      back,
	}

	ch, err := i.backend.ListenMsg()
	if err != nil {
		return nil, err
	}

	go i.parse(ch)
	return i, nil
}

func (i *Ipc) SendCreateMsg(socketId uint32, alg string) error {
	msg, err := makeCreateMsg(socketId, alg)
	if err != nil {
		return err
	}

	return i.backend.SendMsg(msg)
}

func (i *Ipc) SendDropMsg(socketId uint32, ev string) error {
	msg, err := makeDropMsg(socketId, ev)
	if err != nil {
		return err
	}

	return i.backend.SendMsg(msg)
}

func (i *Ipc) SendAckMsg(socketId uint32, ackNo uint32, rtt time.Duration) error {
	msg, err := makeNotifyAckMsg(socketId, ackNo, rtt)
	if err != nil {
		return err
	}

	return i.backend.SendMsg(msg)
}

func (i *Ipc) SendCwndMsg(socketId uint32, cwnd uint32) error {
	msg, err := makeCwndMsg(socketId, cwnd)
	if err != nil {
		return err
	}

	return i.backend.SendMsg(msg)
}

func (i *Ipc) ListenCreateMsg() (chan CreateMsg, error) {
	return i.CreateNotify, nil
}

func (i *Ipc) ListenDropMsg() (chan DropMsg, error) {
	return i.DropNotify, nil
}

func (i *Ipc) ListenAckMsg() (chan AckMsg, error) {
	return i.AckNotify, nil
}

func (i *Ipc) ListenCwndMsg() (chan CwndMsg, error) {
	return i.CwndNotify, nil
}

func (i *Ipc) Close() error {
	return i.backend.Close()
}

func (i *Ipc) parse(msgs chan *capnp.Message) {
	for msg := range msgs {
		if akMsg, err := readAckMsg(msg); err == nil {
			select {
			case i.AckNotify <- akMsg:
			default:
			}
			continue
		}

		if cwMsg, err := readCwndMsg(msg); err == nil {
			select {
			case i.CwndNotify <- cwMsg:
			default:
			}
			continue
		}

		if drMsg, err := readDropMsg(msg); err == nil {
			select {
			case i.DropNotify <- drMsg:
			default:
			}
			continue
		}

		if crMsg, err := readCreateMsg(msg); err == nil {
			select {
			case i.CreateNotify <- crMsg:
			default:
			}
			continue
		}
	}
}
