package udpDataplane

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type windowEntry struct {
	p      *Packet
	t      time.Time
	active bool
}

type window struct {
	mux   sync.RWMutex
	pkts  map[uint32]windowEntry
	order []uint32
}

func (w *window) getOrder() (ord []uint32) {
	w.mux.Lock()
	defer w.mux.Unlock()

	ord = make([]uint32, len(w.order))
	copy(ord, w.order)
	return
}

func makeWindow() *window {
	return &window{
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
	w.mux.Lock()
	defer w.mux.Unlock()

	if e, ok := w.pkts[p.SeqNo]; ok {
		e.t = t
		w.pkts[p.SeqNo] = e
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

func remove(ord []uint32, vals []uint32) (newOrd []uint32) {
	newOrd = ord[:]
	for _, s := range vals {
		for i, v := range newOrd {
			if v == s {
				newOrd = append(newOrd[:i], newOrd[i+1:]...)
				break
			}
		}
	}

	return
}

// sender side
// what cumulative ack has been received
func (w *window) rcvdPkt(
	t time.Time,
	p *Packet,
) (seqNo uint32, rtt time.Duration, err error) {
	seqNo, err = w.start()
	if err != nil {
		return
	}

	w.mux.Lock()
	defer w.mux.Unlock()

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

	// handle SACKs
	removeVals := make([]uint32, 0)
	for i, v := range p.Sack {
		if v {
			seq := p.AckNo + uint32(1460*(i+1))
			e := w.pkts[seq]
			rtt = t.Sub(e.t)
			delete(w.pkts, seq)
			removeVals = append(removeVals, seq)
		}
	}

	w.order = remove(w.order, removeVals)

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
	w.mux.Lock()
	defer w.mux.Unlock()

	for i, e := range w.pkts {
		e.active = false
		w.pkts[i] = e
	}
}

func (w *window) drop(seq uint32) {
	w.mux.Lock()
	defer w.mux.Unlock()

	if ent, ok := w.pkts[seq]; ok {
		ent.active = false
		w.pkts[seq] = ent
	}
}

func (w *window) getNextPkt(newPacket uint32) (seq uint32) {
	w.mux.Lock()
	defer w.mux.Unlock()

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
	w.mux.RLock()
	defer w.mux.RUnlock()

	sz := uint32(0)
	for _, e := range w.pkts {
		if e.active {
			sz += uint32(e.p.Length) + uint32(10)
		}
	}

	return sz
}

func (w *window) start() (uint32, error) {
	w.mux.RLock()
	defer w.mux.RUnlock()

	if len(w.order) == 0 {
		return 0, fmt.Errorf("empty window")
	}

	return w.order[0], nil
}

// receiver side
// what cumulative ack to send
func (w *window) cumAck(start uint32) (uint32, error) {
	w.mux.Lock()
	defer w.mux.Unlock()

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

// receiver side
// the SACK got-packet vector, starting from right after the cumulative ack
// must call cumAck() before this to ensure cumulative ack is correct
func (w *window) getSack(cumAck uint32) (sack []bool) {
	w.mux.RLock()
	defer w.mux.RUnlock()

	sack = make([]bool, 16)
	for _, seq := range w.order {
		sackInd := (seq - cumAck) / 1460
		if sackInd >= uint32(len(sack)) {
			break
		}

		sack[sackInd-1] = true // -1 because we know the cumulative ack hasn't been received
	}

	return
}
