package ipc

import (
	"time"

	flowPattern "ccp/ccpFlow/pattern"
)

// the external serialization interface

type CreateMsg struct {
	socketId uint32
	startSeq uint32
	congAlg  string
}

func (c *CreateMsg) New(sid uint32, startSeq uint32, alg string) {
	c.socketId = sid
	c.startSeq = startSeq
	c.congAlg = alg
}

func (c *CreateMsg) SocketId() uint32 {
	return c.socketId
}

func (c *CreateMsg) StartSeq() uint32 {
	return c.startSeq
}

func (c *CreateMsg) CongAlg() string {
	return c.congAlg
}

func (c *CreateMsg) Serialize() ([]byte, error) {
	return msgWriter(ipcMsg{
		typ:      CREATE,
		socketId: c.socketId,
		u32s:     []uint32{c.startSeq},
		str:      c.congAlg,
	})
}

type MeasureMsg struct {
	socketId uint32
	ackNo    uint32
	rtt      time.Duration
	rin      uint64
	rout     uint64
}

func (m *MeasureMsg) New(
	sid uint32,
	ack uint32,
	t time.Duration,
	rin uint64,
	rout uint64,
) {
	m.socketId = sid
	m.ackNo = ack
	m.rtt = t
	m.rin = rin
	m.rout = rout
}

func (m *MeasureMsg) SocketId() uint32 {
	return m.socketId
}

func (m *MeasureMsg) AckNo() uint32 {
	return m.ackNo
}

func (m *MeasureMsg) Rtt() time.Duration {
	return m.rtt
}

func (m *MeasureMsg) Rin() uint64 {
	return m.rin
}

func (m *MeasureMsg) Rout() uint64 {
	return m.rout
}

func (m *MeasureMsg) Serialize() ([]byte, error) {
	return msgWriter(ipcMsg{
		typ:      MEASURE,
		socketId: m.socketId,
		u32s:     []uint32{m.ackNo, uint32(m.rtt.Nanoseconds())},
		u64s:     []uint64{m.rin, m.rout},
	})
}

type SetMsg struct {
	socketId   uint32
	cwndOrRate uint32
	mode       string
}

func (s *SetMsg) New(sid uint32, cw uint32, m string) {
	s.socketId = sid
	s.cwndOrRate = cw
	s.mode = m
}

func (s *SetMsg) SocketId() uint32 {
	return s.socketId
}

func (s *SetMsg) Set() uint32 {
	return s.cwndOrRate
}

func (s *SetMsg) Mode() string {
	return s.mode
}

func (s *SetMsg) Serialize() ([]byte, error) {
	return msgWriter(ipcMsg{
		typ:      SET,
		socketId: s.socketId,
		u32s:     []uint32{s.cwndOrRate},
		str:      s.mode,
	})
}

type DropMsg struct {
	socketId uint32
	event    string
}

func (d *DropMsg) New(sid uint32, ev string) {
	d.socketId = sid
	d.event = ev
}

func (d *DropMsg) SocketId() uint32 {
	return d.socketId
}

func (d *DropMsg) Event() string {
	return d.event
}

func (d *DropMsg) Serialize() ([]byte, error) {
	return msgWriter(ipcMsg{
		typ:      DROP,
		socketId: d.socketId,
		str:      d.event,
	})
}

type PatternMsg struct {
	socketId uint32
	pattern  *flowPattern.Pattern
}

func (p *PatternMsg) New(sid uint32, t *flowPattern.Pattern) {
	p.socketId = sid
	p.pattern = t
}

func (p *PatternMsg) SocketId() uint32 {
	return p.socketId
}

func (p *PatternMsg) Pattern() *flowPattern.Pattern {
	return p.pattern
}

func (p *PatternMsg) Serialize() ([]byte, error) {
	s, err := serializeSequence(p.pattern.Sequence)
	if err != nil {
		return nil, err
	}

	return msgWriter(ipcMsg{
		typ:      PATTERN,
		socketId: p.socketId,
		u32s:     []uint32{uint32(len(p.pattern.Sequence))},
		str:      string(s),
	})
}

func (i *Ipc) SendCreateMsg(
	socketId uint32,
	startSeq uint32,
	alg string,
) error {
	return i.backend.SendMsg(&CreateMsg{
		socketId: socketId,
		startSeq: startSeq,
		congAlg:  alg,
	})
}

func (i *Ipc) SendMeasureMsg(
	socketId uint32,
	ack uint32,
	rtt time.Duration,
	rin uint64,
	rout uint64,
) error {
	return i.backend.SendMsg(&MeasureMsg{
		socketId: socketId,
		ackNo:    ack,
		rtt:      rtt,
		rin:      rin,
		rout:     rout,
	})
}

func (i *Ipc) SendCwndMsg(socketId uint32, cwnd uint32) error {
	return i.backend.SendMsg(&SetMsg{
		socketId:   socketId,
		cwndOrRate: cwnd,
		mode:       "cwnd",
	})
}

func (i *Ipc) SendRateMsg(socketId uint32, rate uint32) error {
	return i.backend.SendMsg(&SetMsg{
		socketId:   socketId,
		cwndOrRate: rate,
		mode:       "rate",
	})
}

func (i *Ipc) SendDropMsg(socketId uint32, ev string) error {
	return i.backend.SendMsg(&DropMsg{
		socketId: socketId,
		event:    ev,
	})
}

func (i *Ipc) SendPatternMsg(socketId uint32, pattern *flowPattern.Pattern) error {
	return i.backend.SendMsg(&PatternMsg{
		socketId: socketId,
		pattern:  pattern,
	})
}

func (i *Ipc) ListenCreateMsg() (chan CreateMsg, error) {
	return i.CreateNotify, nil
}

func (i *Ipc) ListenDropMsg() (chan DropMsg, error) {
	return i.DropNotify, nil
}

func (i *Ipc) ListenMeasureMsg() (chan MeasureMsg, error) {
	return i.MeasureNotify, nil
}

func (i *Ipc) ListenSetMsg() (chan SetMsg, error) {
	return i.SetNotify, nil
}

func (i *Ipc) ListenPatternMsg() (chan PatternMsg, error) {
	return i.PatternNotify, nil
}
