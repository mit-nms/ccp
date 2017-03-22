package reno

import (
	"ccp/ccpFlow"
	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
)

const pktSize = 1462
const initCwnd = pktSize * 5

// implement ccpFlow.Flow interface
type Reno struct {
	// state
	cwnd    float32
	lastAck uint32

	sockid uint32
	ipc    ipc.SendOnly
}

func (r *Reno) Name() string {
	return "reno"
}

func (r *Reno) Create(socketid uint32, send ipc.SendOnly) {
	r.sockid = socketid
	r.lastAck = 0
	r.cwnd = initCwnd
	r.ipc = send
}

func (r *Reno) Ack(ack uint32) {
	newBytesAcked := float32(ack - r.lastAck)
	// increase cwnd by 1 / cwnd per packet
	r.cwnd += pktSize * (newBytesAcked / r.cwnd)
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
	switch ev {
	case ccpFlow.Isolated:
		r.cwnd /= 2
		if r.cwnd < initCwnd {
			r.cwnd = initCwnd
		}
	case ccpFlow.Complete:
		r.cwnd = initCwnd
	default:
		log.WithFields(log.Fields{
			"event": ev,
		}).Warn("[reno] unknown drop event type")
		return
	}

	log.WithFields(log.Fields{
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
