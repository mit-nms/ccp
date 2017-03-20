package ipc

import (
	"fmt"
	"testing"
	"time"

	"ccp/ipcBackend"
	"ccp/unixsocket"

	log "github.com/Sirupsen/logrus"
)

func reader(b ipcbackend.Backend, ready chan interface{}, done chan error) {
	s, err := SetupWithBackend(b)
	if err != nil {
		done <- err
		return
	}

	inCh, _ := s.ListenAckMsg()
	ready <- struct{}{}
	log.Info("waiting for message")
	ackMsg := <-inCh
	log.Info("got msg")

	if ackMsg.SocketId != 4 || ackMsg.AckNo != 42 {
		done <- fmt.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", ackMsg.SocketId, ackMsg.AckNo, 4, 42)
		return
	}

	s.Close()
	done <- nil
}

func writer(b ipcbackend.Backend, done chan error) {
	s, err := SetupWithBackend(b)
	if err != nil {
		done <- err
		return
	}

	log.Info("writing")
	err = s.SendAckMsg(4, 42)
	if err != nil {
		done <- err
		return
	}

	done <- nil
}

func TestCommunicationUnix(t *testing.T) {
	rdone := make(chan error)
	wdone := make(chan error)
	ready := make(chan interface{})
	go readerUnix(ready, rdone)
	go writerUnix(ready, wdone)

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

func readerUnix(ready chan interface{}, done chan error) {
	b, err := unixsocket.New().SetupListen("ipc-test", 0).SetupFinish()
	if err != nil {
		done <- err
		return
	}

	reader(b, ready, done)
}

func writerUnix(ready chan interface{}, done chan error) {
	<-ready

	b, err := unixsocket.New().SetupSend("ipc-test", 0).SetupFinish()
	if err != nil {
		done <- err
		return
	}

	writer(b, done)
}
