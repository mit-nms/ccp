package mmap

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	capnpMsg "simbackend/capnpMsg"
	"testing"
	"zombiezen.com/go/capnproto2"
)

func setup1(s int) error {
	f, err := os.Create(FILENAME)
	if err != nil {
		return err
	}

	f.Seek(int64(s), 0)
	f.Write([]byte{0})

	f.Close()
	return nil
}

func TestEncodeMsg(t *testing.T) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Error(err)
		return
	}

	foo, err := capnpMsg.NewNotifyAckMsg(seg)
	if err != nil {
		t.Error(err)
		return
	}

	foo.SetSocketId(4)
	foo.SetAckNo(42)

	buf, err := msg.Marshal()
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("marshalled buf: %v", buf)
	copy(buf[8:16], []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00})
	t.Logf("marsh buf, mod: %v", buf)

	decMsg, err := capnp.Unmarshal(buf)
	if err != nil {
		t.Error(err)
		return
	}

	ackMsg, err := capnpMsg.ReadRootNotifyAckMsg(decMsg)
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("socketId: %d, ackNo %d", ackMsg.SocketId(), ackMsg.AckNo())

	if ackMsg.SocketId() != 4 || ackMsg.AckNo() != 42 {
		t.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", ackMsg.SocketId(), ackMsg.AckNo(), 4, 42)
		return
	}
}

func TestCommunicateMsg(t *testing.T) {
	err := setup1(127)
	if err != nil {
		t.Error(err)
		return
	}
	defer os.Remove(FILENAME)

	done := make(chan error)
	wrote := make(chan error)
	ready := make(chan interface{})
	go writerProto(ready, wrote)
	go readerProto(ready, wrote, done)

	err = <-done
	if err != nil {
		t.Error(err)
	}
}

func readerProto(ready chan interface{}, wrote chan error, done chan error) {
	mm, err := Mmap(FILENAME)
	if err != nil {
		done <- err
		return
	}

	ready <- struct{}{}
	err = <-wrote
	if err != nil {
		done <- err
		return
	}

	msg, err := mm.Dec.Decode()
	if err != nil {
		log.Warn(err)
		done <- err
		return
	}

	log.WithFields(log.Fields{
		"numSeg": msg.NumSegments(),
	}).Info("read msg")

	ackMsg, err := capnpMsg.ReadRootNotifyAckMsg(msg)
	if err != nil {
		done <- err
		return
	}

	if ackMsg.SocketId() != 4 || ackMsg.AckNo() != 42 {
		done <- fmt.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", ackMsg.SocketId(), ackMsg.AckNo(), 4, 42)
		return
	}

	done <- nil
	mm.Close()
}

func writerProto(ready chan interface{}, wrote chan error) {
	mm, err := Mmap(FILENAME)
	if err != nil {
		wrote <- err
		return
	}

	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		wrote <- err
		return
	}

	ackMsg, err := capnpMsg.NewRootNotifyAckMsg(seg)
	if err != nil {
		wrote <- err
		return
	}

	ackMsg.SetSocketId(4)
	ackMsg.SetAckNo(42)

	<-ready

	err = mm.Enc.Encode(msg)
	if err != nil {
		wrote <- err
		return
	}

	wrote <- nil
	mm.Close()
}
