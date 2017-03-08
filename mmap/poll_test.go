package mmap

import (
	"os"
	"testing"
	"time"

	"zombiezen.com/go/capnproto2"
)

func TestBadMsg(t *testing.T) {
	mmap, err := Mmap("/tmp/ccp-test")
	if err != nil {
		t.Error(err)
		return
	}

	defer os.Remove("/tmp/ccp-test")

	m := &MMapIpc{
		in:       mmap,
		listenCh: make(chan *capnp.Message),
	}

	e := make(chan error)
	go func() {
		err := m.doDecode()
		e <- err
	}()

	select {
	case err := <-e:
		if err != nil {
			t.Logf("got error: %s", err)
			return
		} else {
			t.Errorf("expected error, got none")
		}
	case msg := <-m.listenCh:
		a, err := readAckMsg(msg)
		if err != nil {
			t.Errorf("expected err, got decode and err: %s", err)
		}

		t.Errorf("expected error, got %v", a)
	}
}

func TestPoll(t *testing.T) {
	mmap, err := Mmap("/tmp/ccp-test")
	if err != nil {
		t.Error(err)
		return
	}

	defer os.Remove("/tmp/ccp-test")

	m := &MMapIpc{
		in:       mmap,
		listenCh: make(chan *capnp.Message),
	}

	go waitToWrite(m)
	go m.pollMmap()

	msg := <-m.listenCh
	ackMsg, err := readAckMsg(msg)
	if err != nil {
		t.Error(err)
		return
	}

	if ackMsg.SocketId != 4 || ackMsg.AckNo != 42 {
		t.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", ackMsg.SocketId, ackMsg.AckNo, 4, 42)
		return
	}

}

func waitToWrite(m *MMapIpc) error {
	msg, err := makeNotifyAckMsg(4, 42)
	if err != nil {
		return err
	}

	enc := capnp.NewEncoder(m.in.mm)
	go func() {
		<-time.After(time.Second)
		enc.Encode(msg)
	}()
	return nil
}
