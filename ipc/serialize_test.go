package ipc

import (
	"testing"
	"time"

	"ccp/ipcBackend"

	log "github.com/sirupsen/logrus"
)

// dummy backend
type MockBackend struct {
	ch chan []byte
}

func NewMockBackend() ipcbackend.Backend {
	return &MockBackend{
		ch: make(chan []byte),
	}
}

func (d *MockBackend) SetupListen(l string, id uint32) ipcbackend.Backend {
	return d
}

func (d *MockBackend) SetupSend(l string, id uint32) ipcbackend.Backend {
	return d
}

func (d *MockBackend) SetupFinish() (ipcbackend.Backend, error) {
	return d, nil
}

func (d *MockBackend) SendMsg(msg ipcbackend.Msg) error {
	s, err := msg.Serialize()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
			"buf": s,
			"msg": msg,
		}).Warn("failed to serialize message")
		return err
	}

	go func() {
		d.ch <- s
	}()

	return nil
}

func (d *MockBackend) Listen() chan []byte {
	ch := make(chan []byte)
	go func() {
		for b := range d.ch {
			ch <- b
		}
	}()

	return ch
}

func (d *MockBackend) Close() error {
	close(d.ch)
	return nil
}

func testSetup() (i *Ipc, err error) {
	back, err := NewMockBackend().SetupFinish()
	if err != nil {
		return
	}

	i, err = SetupWithBackend(back)
	if err != nil {
		return
	}

	return
}

// test values
const testNum uint32 = 42
const testDuration time.Duration = time.Duration(42) * time.Millisecond
const testBigNum uint64 = 42
const testString string = "forty-two"

func TestEncodeMeasureMsg(t *testing.T) {
	i, err := testSetup()
	if err != nil {
		t.Error(err)
		return
	}

	outMsgCh, _ := i.ListenMeasureMsg()
	i.SendMeasureMsg(
		testNum,
		testNum,
		testDuration,
		testBigNum,
		testBigNum,
	)

	select {
	case out := <-outMsgCh:
		if out.SocketId() != testNum ||
			out.AckNo() != testNum ||
			out.Rtt() != testDuration ||
			out.Rin() != testBigNum ||
			out.Rout() != testBigNum {
			t.Errorf(
				"wrong message\ngot (%v, %v, %v, %v, %v)\nexpected (%v, %v, %v, %v, %v)",
				out.SocketId(),
				out.AckNo(),
				out.Rtt(),
				out.Rin(),
				out.Rout(),
				testNum,
				testNum,
				testDuration,
				testBigNum,
				testBigNum,
			)
		}
	case <-time.After(time.Second):
		t.Error("timed out")
	}
}

func TestEncodeCwndMsg(t *testing.T) {
	i, err := testSetup()
	if err != nil {
		t.Error(err)
		return
	}

	outMsgCh, _ := i.ListenSetMsg()
	i.SendCwndMsg(
		testNum,
		testNum,
	)

	select {
	case out := <-outMsgCh:
		if out.SocketId() != testNum ||
			out.Set() != testNum ||
			out.Mode() != "cwnd" {
			t.Errorf(
				"wrong message\ngot (%v, %v, %v)\nexpected (%v, %v, %v)",
				out.SocketId(),
				out.Set(),
				out.Mode(),
				testNum,
				testNum,
				"cwnd",
			)
		}
	case <-time.After(time.Second):
		t.Error("timed out")
	}
}

func TestEncodeRateMsg(t *testing.T) {
	i, err := testSetup()
	if err != nil {
		t.Error(err)
		return
	}

	outMsgCh, _ := i.ListenSetMsg()
	i.SendRateMsg(
		testNum,
		testNum,
	)

	select {
	case out := <-outMsgCh:
		if out.SocketId() != testNum ||
			out.Set() != testNum ||
			out.Mode() != "rate" {
			t.Errorf(
				"wrong message\ngot (%v, %v, %v)\nexpected (%v, %v, %v)",
				out.SocketId(),
				out.Set(),
				out.Mode(),
				testNum,
				testNum,
				"rate",
			)
		}
	case <-time.After(time.Second):
		t.Error("timed out")
	}
}

func TestEncodeCreateMsg(t *testing.T) {
	i, err := testSetup()
	if err != nil {
		t.Error(err)
		return
	}

	outMsgCh, err := i.ListenCreateMsg()
	if err != nil {
		t.Error(err)
		return
	}

	err = i.SendCreateMsg(
		testNum,
		testNum,
		testString,
	)
	if err != nil {
		t.Error(err)
		return
	}

	select {
	case out := <-outMsgCh:
		if out.SocketId() != testNum ||
			out.StartSeq() != testNum ||
			out.CongAlg() != testString {
			t.Errorf(
				"wrong message\ngot (%v, %v, %v)\nexpected (%v, %v, %v)",
				out.SocketId(),
				out.StartSeq(),
				out.CongAlg(),
				testNum,
				testNum,
				testString,
			)
		}
	case <-time.After(time.Second):
		t.Error("timed out")
	}
}

func TestEncodeDropMsg(t *testing.T) {
	i, err := testSetup()
	if err != nil {
		t.Error(err)
		return
	}

	outMsgCh, _ := i.ListenDropMsg()
	i.SendDropMsg(
		testNum,
		testString,
	)

	select {
	case out := <-outMsgCh:
		if out.SocketId() != testNum ||
			out.Event() != testString {
			t.Errorf(
				"wrong message\ngot (%v, %v)\nexpected (%v, %v)",
				out.SocketId(),
				out.Event(),
				testNum,
				testString,
			)
		}
	case <-time.After(time.Second):
		t.Error("timed out")
	}
}
