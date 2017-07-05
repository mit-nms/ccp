package vegas

import (
	"ccp/ccpFlow"
	"ccp/ccpFlow/pattern"
	"ccp/ipc"

	log "github.com/sirupsen/logrus"
)

// implement ccpFlow.Flow interface
type Vegas struct {
	pktSize  float32
	initCwnd float32

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

func (v *Vegas) Create(
	socketid uint32,
	send ipc.SendOnly,
	pktsz uint32,
	startSeq uint32,
	startCwnd uint32,
) {
	v.sockid = socketid
	v.ipc = send
	v.pktSize = float32(pktsz)
	if startSeq == 0 {
		v.lastAck = startSeq
	} else {
		v.lastAck = startSeq - 1
	}
	v.initCwnd = float32(pktsz * 10)
	v.cwnd = float32(pktsz * startCwnd)
	v.baseRTT = 0
	v.alpha = 2
	v.beta = 4

	v.newPattern()
}

func (v *Vegas) GotMeasurement(m ccpFlow.Measurement) {
    if m.Ack < v.lastAck {
        return
    }
	
    RTT := float32(m.Rtt.Seconds())
	if v.baseRTT <= 0 || RTT < v.baseRTT {
		v.baseRTT = RTT
	}
	newBytesAcked := float32(m.Ack - v.lastAck)

	inQueue := (v.cwnd * (RTT - v.baseRTT)) / (RTT * v.pktSize)
	if inQueue <= v.alpha {
		v.cwnd += v.pktSize
	} else if inQueue >= v.beta {
		v.cwnd -= v.pktSize
	}

	v.newPattern()

	log.WithFields(log.Fields{
		"gotAck":      m.Ack,
		"currCwnd":    v.cwnd,
		"currLastAck": v.lastAck,
		"newlyAcked":  newBytesAcked,
		"InQueue":     inQueue,
		"baseRTT":     v.baseRTT,
	}).Info("[vegas] got ack")

	v.lastAck = m.Ack
	return
}

func (v *Vegas) Drop(ev ccpFlow.DropEvent) {
	switch ev {
	case ccpFlow.DupAck:
		v.cwnd -= v.pktSize
	case ccpFlow.Timeout:
		v.cwnd -= v.pktSize
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

	v.newPattern()
}

func (v *Vegas) newPattern() {
	staticPattern, err := pattern.
		NewPattern().
		Cwnd(uint32(v.cwnd)).
		WaitRtts(0.5).
		Report().
		Compile()
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"cwnd": v.cwnd,
		}).Info("make cwnd msg failed")
		return
	}

	err = v.ipc.SendPatternMsg(v.sockid, staticPattern)
	if err != nil {
		log.WithFields(log.Fields{"cwnd": v.cwnd, "name": v.sockid}).Warn(err)
	}
}

func Init() {
	ccpFlow.Register("vegas", func() ccpFlow.Flow {
		return &Vegas{}
	})
}
