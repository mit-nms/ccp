package ipcbackend

import (
	"fmt"
	"os"
	"time"
)

type Msg interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}

type CreateMsg interface {
	Msg
	New(sid uint32, startSeq uint32, alg string)
	SocketId() uint32
	CongAlg() string
	StartSeq() uint32
}

type MeasureMsg interface {
	Msg
	New(
		sid uint32,
		ack uint32,
		rtt time.Duration,
		rin uint64,
		rout uint64,
	)
	SocketId() uint32
	AckNo() uint32
	Rtt() time.Duration
	Rin() uint64
	Rout() uint64
}

type CwndMsg interface {
	Msg
	New(sid uint32, cwnd uint32)
	SocketId() uint32
	Cwnd() uint32
}

type DropMsg interface {
	Msg
	New(sid uint32, ev string)
	SocketId() uint32
	Event() string
}

type Backend interface {
	SetupListen(l string, id uint32) Backend
	SetupSend(l string, id uint32) Backend
	SetupFinish() (Backend, error)

	GetCreateMsg() CreateMsg
	GetMeasureMsg() MeasureMsg
	GetCwndMsg() CwndMsg
	GetDropMsg() DropMsg

	SendMsg(msg Msg) error
	Listen() chan Msg

	Close() error
}

func AddressForListen(loc string, id uint32) (fd string, openFiles string, err error) {
	err = nil
	if id != 0 {
		dirName := fmt.Sprintf("/tmp/ccp-%d", id)
		os.RemoveAll(dirName)
		err = os.MkdirAll(dirName, 0755)
		if err != nil {
			return "", "", err
		}

		fd = fmt.Sprintf("%s/%s", dirName, loc)
		openFiles = dirName
	} else {
		fd = fmt.Sprintf("/tmp/ccp-%s", loc)
		os.RemoveAll(fd)
		openFiles = fd
	}

	return
}

func AddressForSend(loc string, id uint32) (fd string) {
	if id != 0 {
		fd = fmt.Sprintf("/tmp/ccp-%d/%s", id, loc)
	} else {
		fd = fmt.Sprintf("/tmp/ccp-%s", loc)
	}
	return
}
