package unixsocket

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"ccp/capnpMsg"
	"ccp/ipc"

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

func TestCommunication(t *testing.T) {
	rdone := make(chan error)
	wdone := make(chan error)
	ready := make(chan interface{})
	go reader(ready, rdone)
	go writer(ready, wdone)

	done := 0
	var err error
	for {
		select {
		case err = <-rdone:
		case err = <-wdone:
		case <-time.After(time.Second):
			t.Errorf("timed out")
			return
		}
		if err != nil {
			t.Error(err)
			return
		}
		done++
		if done >= 2 {
			break
		}
	}
}

func reader(ready chan interface{}, done chan error) {
	addrOut, err := net.ResolveUnixAddr("unixgram", "/tmp/ccp-test")
	if err != nil {
		done <- err
		return
	}

	in, err := net.ListenUnixgram("unixgram", addrOut)
	if err != nil {
		done <- err
		return
	}

	ready <- struct{}{}

	buf := make([]byte, 1024)
	_, err = in.Read(buf)
	if err != nil {
		done <- err
		return
	}

	msg, err := capnp.Unmarshal(buf)
	if err != nil {
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

	in.Close()
	os.Remove("/tmp/ccp-test")
	done <- nil
}

func writer(ready chan interface{}, done chan error) {
	addrOut, err := net.ResolveUnixAddr("unixgram", "/tmp/ccp-test")
	if err != nil {
		done <- err
		return
	}

	<-ready

	out, err := net.DialUnix("unixgram", nil, addrOut)
	if err != nil {
		done <- err
		return
	}

	s := &SocketIpc{
		in:  nil,
		out: out,
	}

	msg, err := makeNotifyAckMsg(4, 42)
	if err != nil {
		done <- err
		return
	}

	err = s.SendMsg(msg)
	if err != nil {
		done <- err
		return
	}

	done <- nil
}
