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
	f.Create(42, mockIpc, 1462, 0)

	if f.(*Reno).lastAck != 0 || f.(*Reno).sockid != 42 {
		t.Errorf("got \"%v\", expected lastAck=0 and sockid=42", f)
		return
	}

	f.Ack(uint32(292400), time.Second)
	if f.(*Reno).lastAck != 292400 || f.(*Reno).sockid != 42 {
		t.Errorf("got \"%v\", expected lastAck=292400 and sockid=42", f)
		return
	}

	c := <-ipcMockCh
	if c != 43860 {
		t.Errorf("expected cwnd 43860, got %d", c)
		return
	}

	t.Log("isolated drop")
	f.Drop(ccpFlow.Isolated)
	c = <-ipcMockCh
	if c != 21930 {
		t.Errorf("expected cwnd 21930, got %d", c)
		return
	}

	t.Log("complete drop")
	f.Drop(ccpFlow.Complete)
	c = <-ipcMockCh
	if c != 14620 {
		t.Errorf("expected cwnd 14620, got %d", c)
		return
	}
}
