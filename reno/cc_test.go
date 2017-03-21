package reno

import (
	"bytes"
	"testing"

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

	f.Ack(uint32(4386))
	if f.(*Reno).lastAck != 4386 || f.(*Reno).sockid != 42 {
		t.Errorf("got \"%v\", expected lastAck=4386 and sockid=42", f)
		return
	}

	c := <-ipcMockCh
	if c != 9941 {
		t.Errorf("expected cwnd 9941, got %d", c)
		return
	}
}
