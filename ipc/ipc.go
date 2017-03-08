package ipc

import (
	"ccp/ipcBackend"
	"ccp/unixsocket"

	"zombiezen.com/go/capnproto2"
)

type Ipc struct {
	CreateNotify chan CreateMsg
	AckNotify    chan AckMsg
	CwndNotify   chan CwndMsg

	backend ipcbackend.Backend
}

func Setup(sockid uint32) (Ipc, error) {
	back, err := unixsocket.SetupClient(sockid)
	if err != nil {
		return Ipc{}, err
	}

	return SetupWithBackend(back)
}

func SetupWithBackend(back ipcbackend.Backend) (Ipc, error) {
	i := Ipc{
		CreateNotify: make(chan CreateMsg),
		AckNotify:    make(chan AckMsg),
		CwndNotify:   make(chan CwndMsg),
		backend:      back,
	}

	ch, err := i.backend.ListenMsg()
	if err != nil {
		return Ipc{}, err
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

func (i *Ipc) SendAckMsg(socketId uint32, ackNo uint32) error {
	msg, err := makeNotifyAckMsg(socketId, ackNo)
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
		if crMsg, err := readCreateMsg(msg); err == nil {
			i.CreateNotify <- crMsg
			continue
		}

		if akMsg, err := readAckMsg(msg); err == nil {
			i.AckNotify <- akMsg
			continue
		}

		if cwMsg, err := readCwndMsg(msg); err == nil {
			i.CwndNotify <- cwMsg
			continue
		}
	}
}
