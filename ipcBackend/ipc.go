package ipcbackend

import (
	"fmt"
	"os"

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
