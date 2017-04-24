package unixsocket

import (
	"testing"
	"time"

	"ccp/ipcBackend"
)

func TestEncodeAckMsg(t *testing.T) {
	sk := New()
	akMsg := sk.GetAckMsg()
	akMsg.New(4, 42, time.Duration(time.Millisecond))

	buf, err := akMsg.Serialize()
	if err != nil {
		t.Error(err)
		return
	}

	decMsg := parse(buf).(ipcbackend.AckMsg)
	if decMsg.SocketId() != 4 || decMsg.AckNo() != 42 {
		t.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", decMsg.SocketId(), decMsg.AckNo(), 4, 42)
		return
	}
}

func TestEncodeCwndMsg(t *testing.T) {
	sk := New()
	cwMsg := sk.GetCwndMsg()
	cwMsg.New(5, 52)

	buf, err := cwMsg.Serialize()
	if err != nil {
		t.Error(err)
		return
	}

	decMsg := parse(buf).(ipcbackend.CwndMsg)
	if decMsg.SocketId() != 5 || decMsg.Cwnd() != 52 {
		t.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", decMsg.SocketId(), decMsg.Cwnd(), 5, 52)
		return
	}
}

func TestEncodeCreateMsg(t *testing.T) {
	sk := New()
	crMsg := sk.GetCreateMsg()
	crMsg.New(6, 0, "foo")

	buf, err := crMsg.Serialize()
	if err != nil {
		t.Error(err)
		return
	}

	decMsg := parse(buf).(ipcbackend.CreateMsg)
	if decMsg.SocketId() != 6 || decMsg.CongAlg() != "foo" {
		t.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", decMsg.SocketId(), decMsg.CongAlg(), 6, "foo")
		return
	}
}

func TestEncodeDropMsg(t *testing.T) {
	sk := New()
	drMsg := sk.GetDropMsg()
	drMsg.New(7, "bar")

	buf, err := drMsg.Serialize()
	if err != nil {
		t.Error(err)
		return
	}

	decMsg := parse(buf).(ipcbackend.DropMsg)
	if decMsg.SocketId() != 7 || decMsg.Event() != "bar" {
		t.Errorf("wrong message\ngot (%v, %v)\nexpected (%v, %v)", decMsg.SocketId(), decMsg.Event(), 7, "bar")
		return
	}
}

func TestBadEncodeCreateMsg(t *testing.T) {
	sk := New()
	drMsg := sk.GetDropMsg()
	drMsg.New(8, "baz")

	buf, err := drMsg.Serialize()
	if err != nil {
		t.Error(err)
		return
	}

	decMsg := sk.GetCreateMsg()
	err = decMsg.Deserialize(buf)
	if err == nil {
		t.Error("expected error")
		return
	}

	akMsg := sk.GetAckMsg()
	akMsg.New(4, 42, time.Duration(time.Millisecond))

	buf, err = akMsg.Serialize()
	if err != nil {
		t.Error(err)
		return
	}

	decMsg = sk.GetCreateMsg()
	err = decMsg.Deserialize(buf)
	if err == nil {
		t.Error("expected error")
		return
	}
}
