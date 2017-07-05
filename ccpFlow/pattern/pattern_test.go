package pattern

import (
    "testing"
)

func TestFactor(t *testing.T) {
    p, err := NewPattern().Cwnd(42).WaitRtts(1.0).Compile()
    if err != nil {
        t.Error(err)
    }

    if p.err != nil {
        t.Errorf("pattern error %s should be cleared", p.err)
    }

    if sqlen := len(p.Sequence); sqlen != 2 {
        t.Errorf("incorrect pattern length %d", sqlen)
    }
}
