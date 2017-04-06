package vegas

import (
	"time"

	"ccp/ccpFlow"
	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
)

const pktSize = 1462
const initCwnd = pktSize * 5

// implement ccpFlow.Flow interface
type Vegas struct {
	// state
	cwnd    float32
	lastAck uint32

	sockid  uint32
	ipc     ipc.SendOnly
	baseRTT float32
	alpha   float32
	beta    float32
}

func (v *Vegas) Name() string {
	return "vegas"
}

func (v *Vegas) Create(socketid uint32, send ipc.SendOnly) {
	v.sockid = socketid
	v.lastAck = 0
	v.cwnd = initCwnd
	v.ipc = send
	v.baseRTT = 0
	v.alpha = 2
	v.beta = 4
}

func (v *Vegas) Ack(ack uint32, RTT_TD time.Duration) {
	RTT := float32(RTT_TD.Seconds())
	if v.baseRTT <= 0 || RTT < v.baseRTT {
		v.baseRTT = RTT
	}
	newBytesAcked := float32(ack - v.lastAck)

	inQueue := (v.cwnd * (RTT - v.baseRTT)) / (RTT * pktSize)
	if inQueue <= v.alpha {
		v.cwnd += pktSize
	} else if inQueue >= v.beta {
		v.cwnd -= pktSize
	}

	v.notifyCwnd()

	log.WithFields(log.Fields{
		"gotAck":      ack,
		"currCwnd":    v.cwnd,
		"currLastAck": v.lastAck,
		"newlyAcked":  newBytesAcked,
		"InQueue":     inQueue,
		"baseRTT":     v.baseRTT,
	}).Info("[vegas] got ack")

	v.lastAck = ack
	return
}

func (v *Vegas) Drop(ev ccpFlow.DropEvent) {
	switch ev {
	case ccpFlow.Isolated:
		v.cwnd -= pktSize
	case ccpFlow.Complete:
		v.cwnd -= pktSize
	default:
		log.WithFields(log.Fields{
			"event": ev,
		}).Warn("[vegas] unknown drop event type")
		return
	}

	log.WithFields(log.Fields{
		"currCwnd": v.cwnd,
		"event":    ev,
	}).Info("[vegas] drop")

	v.notifyCwnd()
}

func (v *Vegas) notifyCwnd() {
	err := v.ipc.SendCwndMsg(v.sockid, uint32(v.cwnd))
	if err != nil {
		log.WithFields(log.Fields{"cwnd": v.cwnd, "name": v.sockid}).Warn(err)
	}
}

func Init() {
	ccpFlow.Register("vegas", func() ccpFlow.Flow {
		return &Vegas{}
	})
}
