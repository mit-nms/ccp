package ipc

import (
	"testing"
	"time"

	"ccp/ccpFlow/pattern"
)

func BenchmarkEncodeCreateMsg(b *testing.B) {
	i, err := testSetup(false)
	if err != nil {
		b.Error(err)
		return
	}

	outMsgCh, err := i.ListenCreateMsg()
	if err != nil {
		b.Error(err)
		return
	}

	for k := 0; k < b.N; k++ {
		err = i.SendCreateMsg(
			testNum,
			testNum,
			testString,
		)
		if err != nil {
			b.Error(err)
			return
		}

		select {
		case _ = <-outMsgCh:
			continue
		case <-time.After(time.Second):
			b.Error("timed out")
		}
	}
}

func BenchmarkMeasureMsg(b *testing.B) {
	i, err := testSetup(false)
	if err != nil {
		b.Error(err)
		return
	}

	outMsgCh, _ := i.ListenMeasureMsg()
	for k := 0; k < b.N; k++ {
		i.SendMeasureMsg(
			testNum,
			testNum,
			testDuration,
			testBigNum,
			testBigNum,
		)

		select {
		case _ = <-outMsgCh:
			continue
		case <-time.After(time.Second):
			b.Error("timed out")
		}
	}
}

func BenchmarkEncodeDropMsg(b *testing.B) {
	i, err := testSetup(false)
	if err != nil {
		b.Error(err)
		return
	}

	outMsgCh, _ := i.ListenDropMsg()
	for k := 0; k < b.N; k++ {
		i.SendDropMsg(
			testNum,
			testString,
		)

		select {
		case _ = <-outMsgCh:
			continue
		case <-time.After(time.Second):
			b.Error("timed out")
		}
	}
}

func BenchmarkEncodePatternMsg(t *testing.B) {
	i, err := testSetup(false)
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

	for k := 0; k < t.N; k++ {
		err = i.SendPatternMsg(testNum, p)
		if err != nil {
			t.Error(err)
			return
		}

		select {
		case _ = <-outMsgCh:
			continue
		case <-time.After(time.Second):
			t.Error("timed out")
		}
	}
}
