package mmap

import (
	"time"

	"ccp/ipc"
)

type MMapIpc struct {
	in  MM
	out MM

	ipc.BaseIpcLayer
}

func Setup() (ipc.IpcLayer, error) {
	in, err := Mmap("/tmp/ccp-in")
	if err != nil {
		return nil, err
	}

	out, err := Mmap("/tmp/ccp-out")
	if err != nil {
		return nil, err
	}

	return &MMapIpc{
		in:  in,
		out: out,
	}, nil
}

func (m MMapIpc) Close() error {
	m.in.Close()
	m.out.Close()
	return nil
}

func (m *MMapIpc) SendAckMsg(socketId uint32, ackNo uint32) error {
	msg, err := ipc.MakeNotifyAckMsg(socketId, ackNo)
	if err != nil {
		return err
	}

	return m.out.Enc.Encode(msg)
}

func (m MMapIpc) pollMmap() {
	for _ = range time.Tick(time.Microsecond) {
		err := m.doDecode()
		if err != nil {
			continue
		}
	}
}

func (m *MMapIpc) doDecode() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			m.in.mm.reset()
			return
		}
	}()

	msg, err := m.in.Dec.Decode()
	if err != nil {
		m.in.mm.reset()
		return
	}

	err = m.Parse(msg)
	if err != nil {
		m.in.mm.reset()
	}

	return
}
