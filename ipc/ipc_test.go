package ipc

import (
	"fmt"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	b := &BaseIpcLayer{
		AckNotify:  make(chan AckMsg),
		CwndNotify: make(chan CwndMsg),
	}

	ackMsg, err := MakeNotifyAckMsg(4, 568)
	if err != nil {
		t.Error(err)
		return
	}

	cwndMsg, err := MakeCwndMsg(3, 573)
	if err != nil {
		t.Error(err)
		return
	}

	done := make(chan error)
	go expectAck(b, 4, 568, done)
	go expectCwnd(b, 3, 573, done)

	<-time.After(time.Millisecond)

	err = b.Parse(ackMsg)
	if err != nil {
		t.Error(err)
		return
	}

	err = b.Parse(cwndMsg)
	if err != nil {
		t.Error(err)
		return
	}

	for i := 0; i < 2; i++ {
		err = <-done
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func expectAck(b IpcLayer, sid uint32, ack uint32, done chan error) {
	ms, _ := b.ListenAckMsg()

	m := <-ms
	if m.SocketId != sid || m.AckNo != ack {
		done <- fmt.Errorf(
			"incorrect value in ack message read\nexpected (%d, %d)\ngot (%d, %d)",
			sid,
			ack,
			m.SocketId,
			m.AckNo,
		)
	} else {
		done <- nil
	}
}

func expectCwnd(b IpcLayer, sid uint32, cwnd uint32, done chan error) {
	ms, _ := b.ListenCwndMsg()

	m := <-ms
	if m.SocketId != sid || m.Cwnd != cwnd {
		done <- fmt.Errorf(
			"incorrect value in cwnd message read\nexpected (%d, %d)\ngot (%d, %d)",
			sid,
			cwnd,
			m.SocketId,
			m.Cwnd,
		)
	} else {
		done <- nil
	}
}
