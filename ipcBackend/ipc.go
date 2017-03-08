package ipcbackend

import (
	"zombiezen.com/go/capnproto2"
)

type Backend interface {
	SetupListen(l string, id uint32) Backend
	SetupSend(l string, id uint32) Backend
	SetupFinish() (Backend, error)
	SendMsg(msg *capnp.Message) error
	ListenMsg() (chan *capnp.Message, error)
	Close() error
}
