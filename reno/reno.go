package reno

import (
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

	cwnd    float32
	lastAck uint32
    rtt time.Duration
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
	r.initCwnd = float32(pktsz * 10)
	r.cwnd = float32(pktsz * startCwnd)
    r.lastDrop = time.Now()
    r.rtt  = time.Since(r.lastDrop)
	if startSeq == 0 {
		r.lastAck = startSeq
	} else {
		r.lastAck = startSeq - 1
	}

	r.newPattern()
}

func (r *Reno) GotMeasurement(m ccpFlow.Measurement) {
    if m.Ack < r.lastAck {
        return
    }

	newBytesAcked := float32(m.Ack - r.lastAck)
	// increase cwnd by 1 / cwnd per packet
	r.cwnd += float32(r.pktSize) * (newBytesAcked / r.cwnd)
	// notify increased cwnd
	r.newPattern()
    r.rtt = m.Rtt

	log.WithFields(log.Fields{
		"gotAck":      m.Ack,
		"currCwnd":    r.cwnd,
		"currLastAck": r.lastAck,
		"newlyAcked":  newBytesAcked,
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
		if r.cwnd < r.initCwnd {
			r.cwnd = r.initCwnd
		}
	case ccpFlow.Timeout:
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
