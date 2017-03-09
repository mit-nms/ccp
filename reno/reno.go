package reno

import (
	"ccp/ccpFlow"
	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
)

const initCwnd = 15000

// implement ccpFlow.Flow interface
type Reno struct {
	// state
	cwnd    float32
	lastAck uint32
	pktSize uint32

	sockid uint32
	ipc    ipc.SendOnly
}

func (r *Reno) Name() string {
	return "reno"
}

func (r *Reno) Create(socketid uint32, send ipc.SendOnly) {
	r.lastAck = 0
	r.cwnd = initCwnd
	r.pktSize = 1500
	r.ipc = send
}

func (r *Reno) Ack(ack uint32) {
	newBytesAcked := ack - r.lastAck

	increased := false
	for newPktsAcked := newBytesAcked / r.pktSize; newPktsAcked > 0; newPktsAcked -= 1 {
		// increase cwnd by 1 / cwnd per packet
		r.cwnd += 1 / r.cwnd
	}

	if increased {
		// notify increased cwnd
		r.notifyCwnd()
	}

	r.lastAck = ack
	return
}

func (r *Reno) notifyCwnd() {
	err := r.ipc.SendCwndMsg(r.sockid, uint32(r.cwnd))
	if err != nil {
		log.WithFields(log.Fields{"cwnd": r.cwnd, "name": r.sockid}).Warn(err)
	}
}

func init() {
	ccpFlow.Register("reno", func() ccpFlow.Flow {
		return &Reno{}
	})
}
