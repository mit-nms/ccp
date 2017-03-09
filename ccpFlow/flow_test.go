package ccpFlow

import (
	"testing"

	"ccp/ipc"
)

type TestFlow struct{}

func (t *TestFlow) Name() string {
	return "mock"
}

func (t *TestFlow) Create(sockid uint32, send ipc.SendOnly) {
	return
}

func (t *TestFlow) Ack(ack uint32) {
	return
}

func TestRegister(t *testing.T) {
	err := Register("mock", func() Flow { return &TestFlow{} })
	if err != nil {
		t.Error(err)
		return
	}

	rgs := ListRegistered()

	if len(rgs) != 1 {
		t.Error("wrong number of registered algorithms, wanted 1", len(rgs))
		return
	} else if rgs[0] != "mock" {
		t.Error("wrong registered algorithm, wanted \"mock\"", rgs[0])
		return
	}

	f, err := GetFlow(rgs[0])
	if err != nil {
		t.Error(err)
		return
	}

	if f.Name() != "mock" {
		t.Error("wrong registered algorithm, wanted \"mock\"", rgs[0])
		return
	}
}
