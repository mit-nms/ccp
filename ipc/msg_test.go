package ipc

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"ccp/ipcBackend"
	"ccp/unixsocket"
)

func TestParse(t *testing.T) {
	i := Ipc{
		CreateNotify:  make(chan ipcbackend.CreateMsg),
		MeasureNotify: make(chan ipcbackend.MeasureMsg),
		CwndNotify:    make(chan ipcbackend.CwndMsg),
		DropNotify:    make(chan ipcbackend.DropMsg),
		backend:       unixsocket.New(),
	}

	msgs := make(chan ipcbackend.Msg)
	go i.demux(msgs)

	createMsg := i.backend.GetCreateMsg()
	createMsg.New(3, 0, "reno")

	ackMsg := i.backend.GetMeasureMsg()
	ackMsg.New(4, 568, time.Second, 0, 0)

	cwndMsg := i.backend.GetCwndMsg()
	cwndMsg.New(5, 573)

	dropMsg := i.backend.GetDropMsg()
	dropMsg.New(6, "isolated")

	done := make(chan error)
	go expectCreate(i, 3, "reno", done)
	go expectMeasure(i, 4, 568, done)
	go expectCwnd(i, 5, 573, done)
	go expectDrop(i, 6, "isolated", done)

	<-time.After(time.Millisecond)

	msgs <- ackMsg
	msgs <- cwndMsg
	msgs <- createMsg
	msgs <- dropMsg

	for i := 0; i < 4; i++ {
		t.Log(i)
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
				return
			}
		case <-time.After(time.Second):
			t.Error("test timed out")
		}
	}
}

func expectMeasure(b Ipc, sid uint32, ack uint32, done chan error) {
	ms, _ := b.ListenMeasureMsg()

	m := <-ms
	if m.SocketId() != sid || m.AckNo() != ack {
		done <- fmt.Errorf(
			"incorrect value in ack message read\nexpected (%d, %d)\ngot (%d, %d)",
			sid,
			ack,
			m.SocketId(),
			m.AckNo,
		)
	} else {
		done <- nil
	}
}

func expectCwnd(b Ipc, sid uint32, cwnd uint32, done chan error) {
	ms, _ := b.ListenCwndMsg()

	m := <-ms
	if m.SocketId() != sid || m.Cwnd() != cwnd {
		done <- fmt.Errorf(
			"incorrect value in cwnd message read\nexpected (%d, %d)\ngot (%d, %d)",
			sid,
			cwnd,
			m.SocketId(),
			m.Cwnd(),
		)
	} else {
		done <- nil
	}
}

func expectCreate(b Ipc, sid uint32, alg string, done chan error) {
	ms, _ := b.ListenCreateMsg()

	m := <-ms
	if m.SocketId() != sid || m.CongAlg() != alg {
		done <- fmt.Errorf(
			"incorrect value in create message read\nexpected (%d, %s)\ngot (%d, %s)",
			sid,
			alg,
			m.SocketId(),
			m.CongAlg(),
		)
	} else {
		done <- nil
	}
}

func expectDrop(b Ipc, sid uint32, ev string, done chan error) {
	ms, _ := b.ListenDropMsg()

	m := <-ms
	if m.SocketId() != sid || !bytes.Equal([]byte(m.Event()), []byte(ev)) {
		done <- fmt.Errorf(
			"incorrect value in create message read\nexpected (%d, %s)\ngot (%d, %s)",
			sid,
			ev,
			m.SocketId(),
			m.Event(),
		)
	} else {
		done <- nil
	}
}
