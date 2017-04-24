package ipc

import (
	"fmt"
	"time"

	"ccp/ipcBackend"
	"ccp/netlinkipc"
	"ccp/unixsocket"

	log "github.com/Sirupsen/logrus"
)

type Datapath int

const (
	UDP Datapath = iota
	KERNEL
)

type Ipc struct {
	CreateNotify chan ipcbackend.CreateMsg
	AckNotify    chan ipcbackend.AckMsg
	CwndNotify   chan ipcbackend.CwndMsg
	DropNotify   chan ipcbackend.DropMsg

	backend ipcbackend.Backend
}

// handle of IPC to pass to CC implementations
type SendOnly interface {
	SendCwndMsg(socketId uint32, cwnd uint32) error
}

func SetupCcpListen(datapath Datapath) (*Ipc, error) {
	var back ipcbackend.Backend
	var err error

	switch datapath {
	case UDP:
		back, err = unixsocket.New().SetupListen("ccp-in", 0).SetupFinish()
	case KERNEL:
		back, err = netlinkipc.New().SetupListen("", 0).SetupFinish()
	default:
		return nil, fmt.Errorf("unknown datapath")
	}

	if err != nil {
		return nil, err
	}

	return SetupWithBackend(back)
}

func SetupCcpSend(datapath Datapath, sockid uint32) (SendOnly, error) {
	var back ipcbackend.Backend
	var err error

	switch datapath {
	case UDP:
		back, err = unixsocket.New().SetupSend("ccp-out", sockid).SetupFinish()
	case KERNEL:
		back, err = netlinkipc.New().SetupSend("", 0).SetupFinish()
	default:
		return nil, fmt.Errorf("unknown datapath")
	}

	if err != nil {
		return nil, err
	}

	return SetupWithBackend(back)
}

// Setup both sending and receiving
// Only useful for UDP datapath
func SetupCli(sockid uint32) (*Ipc, error) {
	back, err := unixsocket.New().SetupSend("ccp-in", 0).SetupListen("ccp-out", sockid).SetupFinish()
	if err != nil {
		return nil, err
	}

	return SetupWithBackend(back)
}

func SetupWithBackend(back ipcbackend.Backend) (*Ipc, error) {
	i := &Ipc{
		CreateNotify: make(chan ipcbackend.CreateMsg),
		AckNotify:    make(chan ipcbackend.AckMsg),
		CwndNotify:   make(chan ipcbackend.CwndMsg),
		DropNotify:   make(chan ipcbackend.DropMsg),
		backend:      back,
	}

	ch := i.backend.Listen()
	go i.demux(ch)
	return i, nil
}

func (i *Ipc) demux(ch chan ipcbackend.Msg) {
	for m := range ch {
		switch m.(type) {
		case ipcbackend.DropMsg:
			i.DropNotify <- m.(ipcbackend.DropMsg)
		case ipcbackend.CwndMsg:
			i.CwndNotify <- m.(ipcbackend.CwndMsg)
		case ipcbackend.AckMsg:
			i.AckNotify <- m.(ipcbackend.AckMsg)
		case ipcbackend.CreateMsg:
			i.CreateNotify <- m.(ipcbackend.CreateMsg)
		default:
			log.WithFields(log.Fields{
				"msg": m,
			}).Warn("unknown message")
		}
	}
}

func (i *Ipc) SendCwndMsg(socketId uint32, cwnd uint32) error {
	m := i.backend.GetCwndMsg()
	m.New(socketId, cwnd)

	return i.backend.SendMsg(m)
}

func (i *Ipc) SendAckMsg(socketId uint32, ack uint32, rtt time.Duration) error {
	m := i.backend.GetAckMsg()
	m.New(socketId, ack, rtt)

	return i.backend.SendMsg(m)
}

func (i *Ipc) SendCreateMsg(socketId uint32, alg string) error {
	m := i.backend.GetCreateMsg()
	m.New(socketId, alg)

	return i.backend.SendMsg(m)
}

func (i *Ipc) SendDropMsg(socketId uint32, ev string) error {
	m := i.backend.GetDropMsg()
	m.New(socketId, ev)

	return i.backend.SendMsg(m)
}

func (i *Ipc) ListenCreateMsg() (chan ipcbackend.CreateMsg, error) {
	return i.CreateNotify, nil
}

func (i *Ipc) ListenDropMsg() (chan ipcbackend.DropMsg, error) {
	return i.DropNotify, nil
}

func (i *Ipc) ListenAckMsg() (chan ipcbackend.AckMsg, error) {
	return i.AckNotify, nil
}

func (i *Ipc) ListenCwndMsg() (chan ipcbackend.CwndMsg, error) {
	return i.CwndNotify, nil
}

func (i *Ipc) Close() {
	i.backend.Close()
}
