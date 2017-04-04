package reno

import (
	"bytes"
	"testing"
	"time"

	"ccp/ccpFlow"
)

// mock ipc.SendOnly
type MockSendOnly struct {
	ch chan uint32
}

func (m *MockSendOnly) SendCwndMsg(socketId uint32, cwnd uint32) error {
	go func() {
		m.ch <- cwnd
	}()
	return nil
}

func TestReno(t *testing.T) {
	t.Log("init")
	Init()
	f, err := ccpFlow.GetFlow("reno")
	if err != nil {
		t.Error(err)
		return
	}

	if n := f.Name(); !bytes.Equal([]byte(n), []byte("reno")) {
		t.Errorf("got \"%s\", expected \"reno\"", n)
		return
	}

	ipcMockCh := make(chan uint32)
	mockIpc := &MockSendOnly{ch: ipcMockCh}
	f.Create(42, mockIpc)

	if f.(*Reno).lastAck != 0 || f.(*Reno).sockid != 42 {
		t.Errorf("got \"%v\", expected lastAck=0 and sockid=42", f)
		return
	}

	t.Log("acking packets until 7310")
	f.Ack(uint32(7310), time.Second)
	if f.(*Reno).lastAck != 7310 || f.(*Reno).sockid != 42 {
		t.Errorf("got \"%v\", expected lastAck=7310 and sockid=42", f)
		return
	}

	c := <-ipcMockCh
	if c != 8772 {
		t.Errorf("expected cwnd 8772, got %d", c)
		return
	}

	f.Ack(uint32(51170), time.Second)
	if f.(*Reno).lastAck != 51170 || f.(*Reno).sockid != 42 {
		t.Errorf("got \"%v\", expected lastAck=51170 and sockid=42", f)
		return
	}

	c = <-ipcMockCh
	if c != 16082 {
		t.Errorf("expected cwnd 16082, got %d", c)
		return
	}

	t.Log("isolated drop")
	f.Drop(ccpFlow.Isolated)
	c = <-ipcMockCh
	if c != 8041 {
		t.Errorf("expected cwnd 8041, got %d", c)
		return
	}

	t.Log("complete drop")
	f.Drop(ccpFlow.Complete)
	c = <-ipcMockCh
	if c != 7310 {
		t.Errorf("expected cwnd 7310, got %d", c)
		return
	}
}
