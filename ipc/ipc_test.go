package ipc

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"zombiezen.com/go/capnproto2"
)

func TestEncodeAckMsg(t *testing.T) {
	msg, err := makeNotifyAckMsg(4, 42, time.Duration(time.Millisecond))
	if err != nil {
		t.Error(err)
		return
	}

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

	ackMsg, err := readAckMsg(decMsg)
	if err != nil {
		t.Error(err)
		return
	}

	if ackMsg.SocketId != 4 || ackMsg.AckNo != 42 {
		t.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", ackMsg.SocketId, ackMsg.AckNo, 4, 42)
		return
	}
}

func TestEncodeCwndMsg(t *testing.T) {
	msg, err := makeCwndMsg(5, 52)
	if err != nil {
		t.Error(err)
		return
	}

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

	ackMsg, err := readCwndMsg(decMsg)
	if err != nil {
		t.Error(err)
		return
	}

	if ackMsg.SocketId != 5 || ackMsg.Cwnd != 52 {
		t.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", ackMsg.SocketId, ackMsg.Cwnd, 5, 52)
		return
	}
}

func TestBadEncodeCreateMsg(t *testing.T) {
	msg, err := makeCwndMsg(6, 8)
	if err != nil {
		t.Error(err)
		return
	}

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

	_, err = readCreateMsg(decMsg)
	if err == nil {
		t.Error("expected error")
		return
	}

	msg, err = makeNotifyAckMsg(6, 8, time.Duration(time.Millisecond))
	if err != nil {
		t.Error(err)
		return
	}

	buf, err = msg.Marshal()
	if err != nil {
		t.Error(err)
		return
	}

	decMsg, err = capnp.Unmarshal(buf)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = readCreateMsg(decMsg)
	if err == nil {
		t.Error("expected error")
		return
	}
}

func TestEncodeCreateMsg(t *testing.T) {
	msg, err := makeCreateMsg(6, "foo")
	if err != nil {
		t.Error(err)
		return
	}

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

	ackMsg, err := readCreateMsg(decMsg)
	if err != nil {
		t.Error(err)
		return
	}

	if ackMsg.SocketId != 6 || ackMsg.CongAlg != "foo" {
		t.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", ackMsg.SocketId, ackMsg.CongAlg, 6, "foo")
		return
	}
}

func TestEncodeDropMsg(t *testing.T) {
	msg, err := makeDropMsg(7, "isolated")
	if err != nil {
		t.Error(err)
		return
	}

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

	ackMsg, err := readDropMsg(decMsg)
	if err != nil {
		t.Error(err)
		return
	}

	if ackMsg.SocketId != 7 || ackMsg.Event != "isolated" {
		t.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", ackMsg.SocketId, ackMsg.Event, 7, "isolated")
		return
	}
}

func TestParse(t *testing.T) {
	i := Ipc{
		CreateNotify: make(chan CreateMsg),
		AckNotify:    make(chan AckMsg),
		CwndNotify:   make(chan CwndMsg),
		DropNotify:   make(chan DropMsg),
	}

	msgs := make(chan *capnp.Message)
	go i.parse(msgs)

	cwndMsg, err := makeCwndMsg(3, 573)
	if err != nil {
		t.Error(err)
		return
	}

	ackMsg, err := makeNotifyAckMsg(4, 568, time.Duration(time.Millisecond))
	if err != nil {
		t.Error(err)
		return
	}

	createMsg, err := makeCreateMsg(5, "reno")
	if err != nil {
		t.Error(err)
		return
	}

	dropMsg, err := makeDropMsg(6, "isolated")
	if err != nil {
		t.Error(err)
		return
	}

	done := make(chan error)
	go expectAck(i, 4, 568, done)
	go expectCwnd(i, 3, 573, done)
	go expectCreate(i, 5, "reno", done)
	go expectDrop(i, 6, "isolated", done)

	<-time.After(time.Millisecond)

	msgs <- ackMsg
	msgs <- cwndMsg
	msgs <- createMsg
	msgs <- dropMsg

	for i := 0; i < 4; i++ {
		t.Log(i)
		select {
		case err = <-done:
			if err != nil {
				t.Error(err)
				return
			}
		case <-time.After(time.Second):
			t.Error("test timed out")
		}
	}
}

func expectAck(b Ipc, sid uint32, ack uint32, done chan error) {
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

func expectCwnd(b Ipc, sid uint32, cwnd uint32, done chan error) {
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

func expectCreate(b Ipc, sid uint32, alg string, done chan error) {
	ms, _ := b.ListenCreateMsg()

	m := <-ms
	if m.SocketId != sid || m.CongAlg != alg {
		done <- fmt.Errorf(
			"incorrect value in create message read\nexpected (%d, %s)\ngot (%d, %s)",
			sid,
			alg,
			m.SocketId,
			m.CongAlg,
		)
	} else {
		done <- nil
	}
}

func expectDrop(b Ipc, sid uint32, ev string, done chan error) {
	ms, _ := b.ListenDropMsg()

	m := <-ms
	if m.SocketId != sid || !bytes.Equal([]byte(m.Event), []byte(ev)) {
		done <- fmt.Errorf(
			"incorrect value in create message read\nexpected (%d, %s)\ngot (%d, %s)",
			sid,
			ev,
			m.SocketId,
			m.Event,
		)
	} else {
		done <- nil
	}
}
