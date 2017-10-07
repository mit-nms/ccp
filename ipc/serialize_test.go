package ipc

import (
	"testing"
	"time"

	"ccp/ccpFlow/pattern"
	"ccp/ipcBackend"

	log "github.com/sirupsen/logrus"
)

// dummy backend
type MockBackend struct {
	ch    chan []byte
	doLog bool
}

func NewMockBackend(doLog bool) ipcbackend.Backend {
	return &MockBackend{
		ch:    make(chan []byte),
		doLog: doLog,
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

	if d.doLog {
		log.WithFields(log.Fields{
			"msg": s,
		}).Info("sending message")
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

func testSetup(doLog bool) (i *Ipc, err error) {
	back, err := NewMockBackend(doLog).SetupFinish()
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
	i, err := testSetup(true)
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

func TestEncodeCreateMsg(t *testing.T) {
	i, err := testSetup(true)
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
	i, err := testSetup(true)
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

func TestEncodePatternMsg(t *testing.T) {
	i, err := testSetup(true)
	if err != nil {
		t.Error(err)
		return
	}

	outMsgCh, _ := i.ListenPatternMsg()
	p, err := pattern.
		NewPattern().
		Cwnd(testNum).
		WaitRtts(1.0).
		Report().
		Compile()
	if err != nil {
		t.Error(err)
		return
	}

	err = i.SendPatternMsg(testNum, p)
	if err != nil {
		t.Error(err)
		return
	}

	select {
	case out := <-outMsgCh:
		if out.SocketId() != testNum {
			t.Errorf(
				"wrong message\ngot sid (%v)\nexpected (%v)",
				out.SocketId(),
				testNum,
			)
			return
		}

		if len(out.Pattern().Sequence) != 3 {
			t.Errorf(
				"wrong pattern length\ngot %v\nexpected %v",
				len(out.Pattern().Sequence),
				3,
			)
			return
		}

		for i, ev := range out.Pattern().Sequence {
			switch i {
			case 0: // cwnd
				if ev.Type != pattern.SETCWNDABS || ev.Cwnd != testNum {
					t.Errorf(
						"wrong message\ngot event %d (type %v, cwnd %d)\nexpected (%v, %v)",
						i,
						ev.Type,
						ev.Cwnd,
						pattern.SETCWNDABS,
						testNum,
					)
					return
				}
			case 1: // WaitRtts
				if ev.Type != pattern.WAITREL || ev.Factor != 1.0 {
					t.Errorf(
						"wrong message\ngot event %d (type %v, factor %v)\nexpected (%v, %v)",
						i,
						ev.Type,
						ev.Factor,
						pattern.WAITREL,
						1.0,
					)
					return
				}
			case 2: // Report
				if ev.Type != pattern.REPORT {
					t.Errorf(
						"wrong message\ngot event %d type %d\nexpected %v",
						i,
						ev.Type,
						pattern.REPORT,
					)
					return
				}
			}
		}
	case <-time.After(time.Second):
		t.Error("timed out")
	}
}
