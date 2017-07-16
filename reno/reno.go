package reno

import (
	"math"
	"time"

	"ccp/ccpFlow"
	"ccp/ccpFlow/pattern"
	"ccp/ipc"

	log "github.com/sirupsen/logrus"
)

// implement ccpFlow.Flow interface
type Reno struct {
	pktSize  uint32
	initCwnd float32

	ssthresh float32
	cwnd     float32
	lastAck  uint32
	rtt      time.Duration
	lastDrop time.Time

	sockid uint32
	ipc    ipc.SendOnly
}

func (r *Reno) Name() string {
	return "reno"
}

func (r *Reno) Create(
	socketid uint32,
	send ipc.SendOnly,
	pktsz uint32,
	startSeq uint32,
	startCwnd uint32,
) {
	r.sockid = socketid
	r.ipc = send
	r.pktSize = pktsz
	r.ssthresh = 0x7fffffff
	r.initCwnd = float32(pktsz * 10)
	r.cwnd = float32(pktsz * startCwnd)
	r.lastDrop = time.Now()
	r.rtt = time.Since(r.lastDrop)
	if startSeq == 0 {
		r.lastAck = startSeq
	} else {
		r.lastAck = startSeq - 1
	}

	r.newPattern()
}

func (r *Reno) GotMeasurement(m ccpFlow.Measurement) {
	// reordering of messsages
	// if within 10 packets, assume no integer overflow
	if m.Ack < r.lastAck && m.Ack > r.lastAck-r.pktSize*10 {
		return
	}

	// handle integer overflow / sequence wraparound
	var newBytesAcked uint64
	if m.Ack < r.lastAck {
		newBytesAcked = uint64(math.MaxUint32) + uint64(m.Ack) - uint64(r.lastAck)
	} else {
		newBytesAcked = uint64(m.Ack) - uint64(r.lastAck)
	}

	if r.cwnd < r.ssthresh {
		// increase cwnd by 1 per packet
		r.cwnd += float32(newBytesAcked)
	} else {
		// increase cwnd by 1 / cwnd per packet
		r.cwnd += float32(r.pktSize) * (float32(newBytesAcked) / r.cwnd)
	}

	// notify increased cwnd
	r.newPattern()
	r.rtt = m.Rtt

	log.WithFields(log.Fields{
		"gotAck":       m.Ack,
		"currCwndPkts": r.cwnd / float32(r.pktSize),
		"currLastAck":  r.lastAck,
		"newlyAcked":   newBytesAcked,
	}).Info("[reno] got ack")

	r.lastAck = m.Ack
	return
}

func (r *Reno) Drop(ev ccpFlow.DropEvent) {
	if time.Since(r.lastDrop) <= r.rtt {
		return
	}

	r.lastDrop = time.Now()

	oldCwnd := r.cwnd
	switch ev {
	case ccpFlow.DupAck:
		r.cwnd /= 2
		r.ssthresh = r.cwnd
		if r.cwnd < r.initCwnd {
			r.cwnd = r.initCwnd
		}
	case ccpFlow.Timeout:
		r.ssthresh = r.cwnd / 2
		r.cwnd = r.initCwnd
	default:
		log.WithFields(log.Fields{
			"event": ev,
		}).Warn("[reno] unknown drop event type")
		return
	}

	log.WithFields(log.Fields{
		"oldCwnd":  oldCwnd,
		"currCwnd": r.cwnd,
		"event":    ev,
	}).Info("[reno] drop")

	r.newPattern()
}

func (r *Reno) newPattern() {
	staticPattern, err := pattern.
		NewPattern().
		Cwnd(uint32(r.cwnd)).
		WaitRtts(0.5).
		Report().
		Compile()
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"cwnd": r.cwnd,
		}).Info("make cwnd msg failed")
		return
	}

	err = r.ipc.SendPatternMsg(r.sockid, staticPattern)
	if err != nil {
		log.WithFields(log.Fields{"cwnd": r.cwnd, "name": r.sockid}).Warn(err)
	}
}

func Init() {
	ccpFlow.Register("reno", func() ccpFlow.Flow {
		return &Reno{}
	})
}
