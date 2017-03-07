package mmap

import (
	"time"

	capnpMsg "simbackend/capnpMsg"
	"simbackend/ipc"

	"zombiezen.com/go/capnproto2"
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
	// Cap'n Proto! Message passing interface to mmap file
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return err
	}

	notifyAckMsg, err := capnpMsg.NewRootNotifyAckMsg(seg)
	if err != nil {
		return err
	}

	notifyAckMsg.SetSocketId(4)
	notifyAckMsg.SetAckNo(ackNo)

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

	cwnd, err = readCwndMsg(msg)
	if err != nil {
		m.in.mm.reset()
		return
	}

	return
}

func readCwndMsg(msg *capnp.Message) (uint32, error) {
	cwndUpdateMsg, err := capnpMsg.ReadRootSetCwndMsg(msg)
	if err != nil {
		return 0, err
	}

	return cwndUpdateMsg.Cwnd(), nil
}
