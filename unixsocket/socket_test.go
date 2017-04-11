package unixsocket

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

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
	os.RemoveAll("/tmp/ccp-test")
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

	sk := New()
	decMsg := sk.GetAckMsg()
	err = decMsg.Deserialize(buf)
	if err != nil {
		done <- err
		return
	}

	if decMsg.SocketId() != 4 || decMsg.AckNo() != 42 {
		done <- fmt.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", decMsg.SocketId(), decMsg.AckNo(), 4, 42)
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

	akMsg := s.GetAckMsg()
	akMsg.New(4, 42, time.Duration(time.Millisecond))

	err = s.SendMsg(akMsg)
	if err != nil {
		done <- err
		return
	}

	done <- nil
}
