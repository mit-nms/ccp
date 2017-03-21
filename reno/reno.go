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

	log.WithFields(log.Fields{
		"gotAck":      ack,
		"currCwnd":    r.cwnd,
		"currLastAck": r.lastAck,
		"newlyAcked":  newBytesAcked,
	}).Info("got ack notif")

	// increase cwnd by 1 / cwnd per packet
	r.cwnd += newBytesAcked * (newBytesAcked / r.cwnd)
	// notify increased cwnd
	r.notifyCwnd()

	r.lastAck = ack

	log.WithFields(log.Fields{
		"currCwnd": r.cwnd,
	}).Info("updated")
	return
}

func (r *Reno) notifyCwnd() {
	err := r.ipc.SendCwndMsg(r.sockid, uint32(r.cwnd))
	if err != nil {
		log.WithFields(log.Fields{"cwnd": r.cwnd, "name": r.sockid}).Warn(err)
	}
	log.WithFields(log.Fields{"cwnd": r.cwnd, "name": r.sockid}).Info("update cwnd")
}

func Init() {
	log.Info("registering reno")
	ccpFlow.Register("reno", func() ccpFlow.Flow {
		return &Reno{}
	})
}
