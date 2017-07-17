package bbr

import (
	"math"
	"time"

	"ccp/ccpFlow"
	"ccp/ccpFlow/pattern"
	"ccp/ipc"

	log "github.com/sirupsen/logrus"
)

// implement ccpFlow.Flow interface
type BBR struct {
	pktSize  uint32

	lastAck    uint32
	rtt        time.Duration
	lastDrop   time.Time
	lastUpdate time.Time
	rcv_rate   float32
	wait_time  time.Duration

	sockid uint32
	ipc    ipc.SendOnly
}

func (b *BBR) Name() string {
	return "bbr"
}

func (b *BBR) Create(
	socketid uint32,
	send ipc.SendOnly,
	pktsz uint32,
	startSeq uint32,
	startCwnd uint32,
) {
	b.sockid = socketid
	b.ipc = send
	b.pktSize = pktsz
	b.rcv_rate = float32(b.pktSize*100)
	b.lastDrop = time.Now()
	b.lastUpdate = time.Now()
	b.rtt = time.Since(b.lastDrop)
	if startSeq == 0 {
		b.lastAck = startSeq
	} else {
		b.lastAck = startSeq - 1
	}
	b.wait_time = 160*time.Millisecond
	b.sendPattern(0.95*b.rcv_rate, b.wait_time/8)

}

func (b *BBR) GotMeasurement(m ccpFlow.Measurement) {
    // Ignore out of order netlink messages
    // Happens sometimes when the reporting interval is small
	// If within 10 packets, assume no integer overflow
	if m.Ack < b.lastAck && m.Ack > b.lastAck-b.pktSize*10 {
		return
	}

	if b.rcv_rate < float32(m.Rout) {
		b.rcv_rate = float32(m.Rout)
	}

	// Handle integer overflow / sequence wraparound
	var newBytesAcked uint64
	if m.Ack < b.lastAck {
		newBytesAcked = uint64(math.MaxUint32) + uint64(m.Ack) - uint64(b.lastAck)
	} else {
		newBytesAcked = uint64(m.Ack) - uint64(b.lastAck)
	}

    acked := newBytesAcked

    if time.Since(b.lastUpdate) >= b.wait_time {
		b.sendPattern(0.95*b.rcv_rate, b.wait_time/8)
		b.lastUpdate=time.Now()
	}

	b.rtt = m.Rtt

	log.WithFields(log.Fields{
		"gotAck":       m.Ack,
		"currRate":     b.rcv_rate,
		"currLastAck":  b.lastAck,
		"newlyAcked":   acked,
	}).Info("[bbr] got ack")

	b.lastAck = m.Ack
	return
}

func (b *BBR) Drop(ev ccpFlow.DropEvent) {
	if time.Since(b.lastDrop) <= b.rtt {
		return
	}

	oldRate := b.rcv_rate
	log.WithFields(log.Fields{
		"time since last drop": time.Since(b.lastDrop),
		"rtt": b.rtt,
	}).Info("[bbr] got drop")

	b.lastDrop = time.Now()

	switch ev {
	case ccpFlow.DupAck:
		b.rcv_rate /=2
	case ccpFlow.Timeout:
		b.rcv_rate = float32(b.pktSize * 100)
	default:
		log.WithFields(log.Fields{
			"event": ev,
		}).Warn("[bbr] unknown drop event type")
		return
	}

	b.lastUpdate = time.Now()
	b.wait_time = 160*time.Millisecond
	b.sendPattern(0.95*b.rcv_rate, b.wait_time/8)
	
	log.WithFields(log.Fields{
		"oldRate": oldRate,
		"newRate": b.rcv_rate,
		"event":   ev,
	}).Info("[bbr] drop")
}

func (b *BBR) sendPattern(rcv_rate float32, wait_time time.Duration) {
		pattern, err := pattern.
		NewPattern().
		Rate(1.25*rcv_rate).
		Wait(wait_time).
		Report().
		Rate(0.75*rcv_rate).
		Wait(wait_time).
		Report().
		Rate(rcv_rate).
		Wait(wait_time).
		Report().
		Rate(rcv_rate).
		Wait(wait_time).
		Report().
		Rate(rcv_rate).
		Wait(wait_time).
		Report().
		Rate(rcv_rate).
		Wait(wait_time).
		Report().
		Rate(rcv_rate).
		Wait(wait_time).
		Report().
		Rate(rcv_rate).
		Wait(wait_time).
		Report().
		Compile()
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"rate": b.rcv_rate,
		}).Info("make cwnd msg failed")
		return
	}
	err = b.ipc.SendPatternMsg(b.sockid, pattern)
	if err != nil {
		log.WithFields(log.Fields{"rate": b.rcv_rate, "name": b.sockid}).Warn(err)
	}
}

func Init() {
	ccpFlow.Register("bbr", func() ccpFlow.Flow {
		return &BBR{}
	})
}
