package reno

import (
	"time"

	"ccp/ccpFlow"
	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
)

// implement ccpFlow.Flow interface
type Reno struct {
	pktSize  uint32
	initCwnd float32

	cwnd    float32
	lastAck uint32

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
	if startSeq == 0 {
		r.lastAck = startSeq
	} else {
		r.lastAck = startSeq - 1
	}
}

func (r *Reno) Ack(ack uint32, rtt time.Duration) {
	newBytesAcked := float32(ack - r.lastAck)
	// increase cwnd by 1 / cwnd per packet
	r.cwnd += float32(r.pktSize) * (newBytesAcked / r.cwnd)
	// notify increased cwnd
	r.notifyCwnd()

	log.WithFields(log.Fields{
		"gotAck":      ack,
		"currCwnd":    r.cwnd,
		"currLastAck": r.lastAck,
		"newlyAcked":  newBytesAcked,
	}).Info("[reno] got ack")

	r.lastAck = ack
	return
}

func (r *Reno) Drop(ev ccpFlow.DropEvent) {
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

	r.notifyCwnd()
}

func (r *Reno) notifyCwnd() {
	err := r.ipc.SendCwndMsg(r.sockid, uint32(r.cwnd))
	if err != nil {
		log.WithFields(log.Fields{"cwnd": r.cwnd, "name": r.sockid}).Warn(err)
	}
}

func Init() {
	ccpFlow.Register("reno", func() ccpFlow.Flow {
		return &Reno{}
	})
}
