package vegas

import (
	"math"

	"ccp/ccpFlow"
	"ccp/ccpFlow/pattern"
	"ccp/ipc"

	log "github.com/sirupsen/logrus"
)

// implement ccpFlow.Flow interface
type Vegas struct {
	pktSize  uint32
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
	v.pktSize = pktsz
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
	// reordering of messsages
	// if within 10 packets, assume no integer overflow
	if m.Ack < v.lastAck && m.Ack > v.lastAck-v.pktSize*10 {
		return
	}

	// handle integer overflow / sequence wraparound
	var newBytesAcked uint64
	if m.Ack < v.lastAck {
		newBytesAcked = uint64(math.MaxUint32) + uint64(m.Ack) - uint64(v.lastAck)
	} else {
		newBytesAcked = uint64(m.Ack) - uint64(v.lastAck)
	}

	RTT := float32(m.Rtt.Seconds())
	if v.baseRTT <= 0 || RTT < v.baseRTT {
		v.baseRTT = RTT
	}

	inQueue := (v.cwnd * (RTT - v.baseRTT)) / (RTT * float32(v.pktSize))
	if inQueue <= v.alpha {
		v.cwnd += float32(v.pktSize)
	} else if inQueue >= v.beta {
		v.cwnd -= float32(v.pktSize)
	}

	v.newPattern()

	log.WithFields(log.Fields{
		"gotAck":      m.Ack,
		"currCwnd":    v.cwnd,
		"currLastAck": v.lastAck,
		"newlyAcked":  newBytesAcked,
		"InQueue":     inQueue,
		"baseRTT":     v.baseRTT,
		"loss":        m.Loss,
	}).Info("[vegas] got ack")

	v.lastAck = m.Ack
	return
}

func (v *Vegas) Drop(ev ccpFlow.DropEvent) {
	switch ev {
	case ccpFlow.DupAck:
		v.cwnd -= float32(v.pktSize)
	case ccpFlow.Timeout:
		v.cwnd -= float32(v.pktSize)
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
