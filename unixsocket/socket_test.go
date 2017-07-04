package unixsocket

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

// mock message implementing ipcbackend.Msg
type MockMsg struct {
	b string
}

func (m MockMsg) Serialize() ([]byte, error) {
	return []byte(m.b), nil
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

	if string(buf[:3]) != "foo" {
		done <- fmt.Errorf("wrong message\ngot (%s)\nexpected (foo)", buf)
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

	err = s.SendMsg(&MockMsg{b: "foo"})
	if err != nil {
		done <- err
		return
	}

	done <- nil
}
