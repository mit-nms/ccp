package mmap

import (
	"fmt"
	"os"
	"testing"

	"ccp/capnpMsg"
	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
	"zombiezen.com/go/capnproto2"
)

func makeNotifyAckMsg(socketId uint32, ackNo uint32) (*capnp.Message, error) {
	// Cap'n Proto! Message passing interface to mmap file
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	notifyAckMsg, err := capnpMsg.NewRootUIntMsg(seg)
	if err != nil {
		return nil, err
	}

	notifyAckMsg.SetType(capnpMsg.MsgType_ack)
	notifyAckMsg.SetSocketId(socketId)
	notifyAckMsg.SetVal(ackNo)

	return msg, nil
}

func readAckMsg(msg *capnp.Message) (ipc.AckMsg, error) {
	ackMsg, err := capnpMsg.ReadRootUIntMsg(msg)
	if err != nil {
		return ipc.AckMsg{}, err
	}

	if ackMsg.Type() != capnpMsg.MsgType_ack {
		return ipc.AckMsg{}, fmt.Errorf("Message not of type Ack: %v", ackMsg.Type())
	}

	return ipc.AckMsg{SocketId: ackMsg.SocketId(), AckNo: ackMsg.Val()}, nil
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

	ackMsg, err := readAckMsg(msg)
	if err != nil {
		done <- err
		return
	}

	if ackMsg.SocketId != 4 || ackMsg.AckNo != 42 {
		done <- fmt.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", ackMsg.SocketId, ackMsg.AckNo, 4, 42)
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

	msg, err := makeNotifyAckMsg(4, 42)
	if err != nil {
		wrote <- err
		return
	}

	<-ready

	err = mm.Enc.Encode(msg)
	if err != nil {
		wrote <- err
		return
	}

	wrote <- nil
	mm.Close()
}
