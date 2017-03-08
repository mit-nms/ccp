package ipcbackend

import (
	"zombiezen.com/go/capnproto2"
)

type Backend interface {
	SendMsg(msg *capnp.Message) error
	ListenMsg() (chan *capnp.Message, error)
	Close() error
}
