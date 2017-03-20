package udpDataplane

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
)

type windowEntry struct {
	p      *Packet
	t      time.Time
	active bool
}

type window struct {
	pkts  map[uint32]windowEntry
	order []uint32
}

func makeWindow() window {
	return window{
		pkts:  make(map[uint32]windowEntry),
		order: make([]uint32, 0),
	}
}

func insertIndex(list []uint32, it uint32) int {
	if len(list) == 0 {
		return len(list)
	}

	if it < list[0] {
		return 0
	}

	return 1 + insertIndex(list[1:], it)
}

// both sides
func (w *window) addPkt(t time.Time, p *Packet) {
	if e, ok := w.pkts[p.SeqNo]; ok {
		e.t = t
		return
	}

	ind := insertIndex(w.order, p.SeqNo)

	ent := windowEntry{
		p:      p,
		t:      t,
		active: true,
	}
	w.pkts[p.SeqNo] = ent

	// insert into slice at index
	w.order = append(w.order, 0)
	copy(w.order[ind+1:], w.order[ind:])
	w.order[ind] = p.SeqNo

	if len(w.order) != len(w.pkts) {
		log.WithFields(log.Fields{
			"order": w.order,
			"pkts":  w.pkts,
		}).Panic("window out of sync")
	}
}

// sender side
// what cumulative ack has been received
func (w *window) rcvdPkt(
	t time.Time,
	p *Packet,
) (seqNo uint32, rtt time.Duration) {
	seqNo, _ = w.start()
	ind := 0
	for i, seq := range w.order {
		ind = i + 1
		if seq < p.AckNo {
			rtt = t.Sub(w.pkts[seq].t)
			seqNo = seq + uint32(w.pkts[seq].p.Length)
			delete(w.pkts, seq)
			continue
		}
		ind -= 1
		break
	}

	w.order = w.order[ind:]
	if len(w.order) != len(w.pkts) {
		log.WithFields(log.Fields{
			"ind":   ind,
			"order": w.order,
			"pkts":  w.pkts,
		}).Panic("window out of sync")
	}

	return
}

func (w *window) timeout() {
	for i, e := range w.pkts {
		e.active = false
		w.pkts[i] = e
	}
}

func (w *window) drop(seq uint32) {
	if ent, ok := w.pkts[seq]; ok {
		ent.active = false
		w.pkts[seq] = ent
	}
}

func (w *window) getNextPkt(newPacket uint32) (seq uint32) {
	for _, s := range w.order {
		e, _ := w.pkts[s]
		if !e.active {
			e.active = true
			w.pkts[s] = e
			return s
		}
	}

	return newPacket
}

// window size in bytes
func (w *window) size() uint32 {
	sz := uint32(0)
	for _, e := range w.pkts {
		if e.active {
			sz += uint32(e.p.Length) + uint32(10)
		}
	}

	return sz
}

func (w *window) start() (uint32, error) {
	if len(w.order) == 0 {
		return 0, fmt.Errorf("empty window")
	}

	return w.order[0], nil
}

// receiver side
// what cumulative ack to send
func (w *window) cumAck(start uint32) (uint32, error) {
	if len(w.order) == 0 {
		return 0, fmt.Errorf("empty window")
	}

	cumAck := start
	for len(w.order) > 0 && cumAck == w.order[0] {
		w.order = w.order[1:]
		nextCumAck := uint32(w.pkts[cumAck].p.Length)
		delete(w.pkts, cumAck)
		cumAck += nextCumAck
	}

	if len(w.order) != len(w.pkts) {
		log.WithFields(log.Fields{
			"order": w.order,
			"pkts":  w.pkts,
		}).Panic("window out of sync")
	}

	return cumAck, nil
}
