package reno

import (
	"bytes"
	"testing"
	"time"

	"ccp/ccpFlow"
	flowPattern "ccp/ccpFlow/pattern"
)

// mock ipc.SendOnly
type MockSendOnly struct {
	ch chan *flowPattern.Pattern
}

func (m *MockSendOnly) SendPatternMsg(socketId uint32, pattern *flowPattern.Pattern) error {
	go func() {
		m.ch <- pattern
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

	ipcMockCh := make(chan *flowPattern.Pattern)
	mockIpc := &MockSendOnly{ch: ipcMockCh}
	f.Create(42, mockIpc, 1462, 0, 10)
	<-ipcMockCh // ignore the first initial cwnd set

	if f.(*Reno).lastAck != 0 || f.(*Reno).sockid != 42 {
		t.Errorf("got \"%v\", expected lastAck=0 and sockid=42", f)
		return
	}

	f.GotMeasurement(ccpFlow.Measurement{
		Ack: uint32(292400),
		Rtt: time.Microsecond,
	})
	if f.(*Reno).lastAck != 292400 || f.(*Reno).sockid != 42 {
		t.Errorf("got \"%v\", expected lastAck=292400 and sockid=42", f)
		return
	}

	p := <-ipcMockCh
	if len(p.Sequence) != 3 {
		t.Errorf("expected sequence length 3, got %d", len(p.Sequence))
		return
	} else if p.Sequence[0].Type != flowPattern.SETCWNDABS {
		t.Errorf("expected event type %d, got %d", flowPattern.SETCWNDABS, p.Sequence[0].Type)
		return
	}

	t.Log("isolated drop")
	f.Drop(ccpFlow.DupAck)
	p = <-ipcMockCh
	if len(p.Sequence) != 3 {
		t.Errorf("expected sequence length 2, got %d", len(p.Sequence))
		return
	} else if p.Sequence[0].Type != flowPattern.SETCWNDABS {
		t.Errorf("expected event type %d, got %d", flowPattern.SETCWNDABS, p.Sequence[0].Type)
		return
	}

	t.Log("complete drop")
	f.Drop(ccpFlow.Timeout)
	p = <-ipcMockCh
	if len(p.Sequence) != 3 {
		t.Errorf("expected sequence length 2, got %d", len(p.Sequence))
		return
	} else if p.Sequence[0].Type != flowPattern.SETCWNDABS {
		t.Errorf("expected event type %d, got %d", flowPattern.SETCWNDABS, p.Sequence[0].Type)
		return
	} else if p.Sequence[0].Cwnd != 14620 {
		t.Errorf("expected cwnd 14620, got %d", p.Sequence[0].Cwnd)
		return
	}
}
