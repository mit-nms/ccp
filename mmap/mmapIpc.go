package mmap

import (
	"time"

	"ccp/ipc"
)

type MMapIpc struct {
	in  MM
	out MM
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

	return MMapIpc{
		in:  in,
		out: out,
	}, nil
}

func (m MMapIpc) Close() error {
	m.in.Close()
	m.out.Close()
	return nil
}

func (m MMapIpc) Send(socketId uint32, ackNo uint32) error {
	msg, err := ipc.NotifyAckMsg(socketId, ackNo)
	if err != nil {
		return err
	}

	return m.out.Enc.Encode(msg)
}

func (m MMapIpc) Listen() (chan uint32, error) {
	gotMessage := make(chan uint32)
	go m.pollMmap(gotMessage)
	return gotMessage, nil
}

func (m MMapIpc) pollMmap(gotMsg chan uint32) {
	for _ = range time.Tick(time.Microsecond) {
		cwnd, err := m.doDecode()
		if err != nil {
			continue
		}

		gotMsg <- cwnd
	}
}

func (m MMapIpc) doDecode() (cwnd uint32, err error) {
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

	cwnd, err = ipc.ReadCwndMsg(msg)
	if err != nil {
		m.in.mm.reset()
		return
	}

	return
}
