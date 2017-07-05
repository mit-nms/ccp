package pattern

import (
	"fmt"
	"time"
)

type Pattern struct {
	Sequence []PatternEvent
	err      error
}

type PatternEventType uint8

const (
	SETRATEABS PatternEventType = iota
	SETCWNDABS
	SETRATEREL
	WAITABS
	WAITREL
	REPORT
)

type PatternEvent struct {
	Type     PatternEventType
	Duration time.Duration
	Cwnd     uint32
	Rate     float32
	Factor   float32
}

func NewPattern() *Pattern {
	return &Pattern{
		Sequence: make([]PatternEvent, 0),
	}
}

func (p *Pattern) Cwnd(cwnd uint32) *Pattern {
	if p.err != nil {
		return p
	}

	p.Sequence = append(p.Sequence, PatternEvent{
		Type: SETCWNDABS,
		Cwnd: cwnd,
	})

	return p
}

func (p *Pattern) Rate(rate float32) *Pattern {
	if p.err != nil {
		return p
	}

	p.Sequence = append(p.Sequence, PatternEvent{
		Type: SETRATEABS,
		Rate: rate,
	})

	return p
}

func (p *Pattern) RelativeRate(factor float32) *Pattern {
	if p.err != nil {
		return p
	}

	p.Sequence = append(p.Sequence, PatternEvent{
		Type:   SETRATEREL,
		Factor: factor,
	})

	return p
}

func (p *Pattern) Wait(wait time.Duration) *Pattern {
	if p.err != nil {
		return p
	}

	if len(p.Sequence) == 0 {
		p.err = fmt.Errorf("wait cannot start pattern")
		return p
	}

	p.Sequence = append(p.Sequence, PatternEvent{
		Type:     WAITABS,
		Duration: wait,
	})

	return p
}

func (p *Pattern) WaitRtts(factor float32) *Pattern {
	if p.err != nil {
		return p
	}

	p.Sequence = append(p.Sequence, PatternEvent{
		Type:   WAITREL,
		Factor: factor,
	})

	return p
}

func (p *Pattern) Report() *Pattern {
	if p.err != nil {
		return p
	}

	p.Sequence = append(p.Sequence, PatternEvent{
		Type: REPORT,
	})

	return p
}

func (p *Pattern) Compile() (*Pattern, error) {
	if len(p.Sequence) == 1 {
		// add a long wait after
		// this will function as a one-off set
		p = p.Wait(time.Second)
	}

	if p.err != nil {
		return nil, p.err
	}

	return p, nil
}
