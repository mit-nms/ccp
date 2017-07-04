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
	Value    uint32
}

func NewPattern() *Pattern {
	return &Pattern{
		Sequence: make([]PatternEvent, 0),
	}
}

func (p *Pattern) AbsoluteRate(rate float32) *Pattern {
	if p.err != nil {
		return p
	}

	p.Sequence = append(p.Sequence, PatternEvent{
		Type:  SETRATEABS,
		Value: uint32(rate * 100),
	})

	return p
}

func (p *Pattern) RelativeRate(rate float32) *Pattern {
	if p.err != nil {
		return p
	}

	p.Sequence = append(p.Sequence, PatternEvent{
		Type:  SETRATEREL,
		Value: uint32(rate * 100),
	})

	return p
}

func (p *Pattern) AbsoluteCwnd(cwnd uint32) *Pattern {
	if p.err != nil {
		return p
	}

	p.Sequence = append(p.Sequence, PatternEvent{
		Type:  SETCWNDABS,
		Value: cwnd,
	})

	return p
}

func (p *Pattern) AbsoluteWait(wait time.Duration) *Pattern {
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

func (p *Pattern) RelativeWait(waitFrac float32) *Pattern {
	if p.err != nil {
		return p
	}

	p.Sequence = append(p.Sequence, PatternEvent{
		Type:  WAITREL,
		Value: uint32(waitFrac * 100),
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
	if p.err != nil {
		return nil, p.err
	}

	return p, nil
}
