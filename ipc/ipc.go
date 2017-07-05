package ipc

import (
	"fmt"

	flowPattern "ccp/ccpFlow/pattern"
	"ccp/ipcBackend"
	"ccp/netlinkipc"
	"ccp/unixsocket"
)

// setup and teardown logic

type Datapath int

const (
	UNIX Datapath = iota
	NETLINK
)

type Ipc struct {
	CreateNotify  chan CreateMsg
	MeasureNotify chan MeasureMsg
	DropNotify    chan DropMsg
	PatternNotify chan PatternMsg

	backend ipcbackend.Backend
}

// handle of IPC to pass to CC implementations
type SendOnly interface {
	SendPatternMsg(socketId uint32, pattern *flowPattern.Pattern) error
}

func SetupCcpListen(datapath Datapath) (*Ipc, error) {
	var back ipcbackend.Backend
	var err error

	switch datapath {
	case UNIX:
		back, err = unixsocket.New().SetupListen("ccp-in", 0).SetupFinish()
	case NETLINK:
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
	case UNIX:
		back, err = unixsocket.New().SetupSend("ccp-out", sockid).SetupFinish()
	case NETLINK:
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
		CreateNotify:  make(chan CreateMsg),
		MeasureNotify: make(chan MeasureMsg),
		DropNotify:    make(chan DropMsg),
		PatternNotify: make(chan PatternMsg),
		backend:       back,
	}

	ch := i.backend.Listen()
	go i.demux(ch)
	return i, nil
}

func (i *Ipc) Close() error {
	return i.backend.Close()
}
