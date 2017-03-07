package mmap

import (
	"bytes"
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"testing"
	capnpMsg "udpDataplane/capnpMsg"
	"zombiezen.com/go/capnproto2"
)

func TestBadMsg(t *testing.T) {
	b := bytes.Repeat([]byte{0}, 24)
	buf := bytes.NewBuffer(b)
	sid, ackno, err := doDecode(buf)
	if err == nil {
		t.Errorf("expected errors, got socketId: %d, ackNo %d", sid, ackno)
	}
}

func TestPoll(t *testing.T) {
	b := new(bytes.Buffer)
	go waitToWrite(b)
	for _ = range time.Tick(time.Microsecond) {
		sid, ackno, err := doDecode(b)
		if err != nil {
			continue
		}

		if sid != 4 || ackno != 42 {
			t.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", sid, ackno, 4, 42)
			return
		}
		break
	}
}

func waitToWrite(b *bytes.Buffer) error {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return err
	}

	foo, err := capnpMsg.NewRootNotifyAckMsg(seg)
	if err != nil {
		return err
	}

	foo.SetSocketId(4)
	foo.SetAckNo(42)

	dec := capnp.NewEncoder(b)
	go func() {
		<-time.After(time.Second)
		dec.Encode(msg)
	}()
	return nil
}

func doDecode(buf *bytes.Buffer) (sockId uint32, ackno uint32, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			return
		}
	}()

	msg, err := capnp.NewDecoder(buf).Decode()
	if err != nil {
		return
	}

	ack, err := capnpMsg.ReadRootNotifyAckMsg(msg)
	if err != nil {
		return
	}

	return ack.SocketId(), ack.AckNo(), err
}

func TestEncodeMsg(t *testing.T) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Error(err)
		return
	}

	foo, err := capnpMsg.NewRootNotifyAckMsg(seg)
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

	if ackMsg.SocketId() != 4 || ackMsg.AckNo() != 42 {
		t.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", ackMsg.SocketId(), ackMsg.AckNo(), 4, 42)
		return
	}
}

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
