package cubic

//Pseudo Code from https://pdfs.semanticscholar.org/4e8f/00ffc07c77340ba4121b36585754f8b8379a.pdf

import (
	"math"
	"time"

	"ccp/ccpFlow"
	"ccp/ccpFlow/pattern"
	"ccp/ipc"

	log "github.com/sirupsen/logrus"
)

// implement ccpFlow.Flow interface
type Cubic struct {
	pktSize  uint32
	initCwnd float64

	cwnd     float64
	lastAck  uint32
	lastDrop time.Time
	rtt      time.Duration
	sockid   uint32
	ipc      ipc.SendOnly

	//state for cubic
	ssthresh         float64
	cwnd_cnt         float64
	tcp_friendliness bool
	BETA             float64
	fast_convergence bool
	C                float64
	Wlast_max        float64
	epoch_start      float64
	origin_point     float64
	dMin             float64
	Wtcp             float64
	K                float64
	ack_cnt          float64
	cnt              float64
}

func (c *Cubic) Name() string {
	return "cubic"
}

func (c *Cubic) Create(
	socketid uint32,
	send ipc.SendOnly,
	pktsz uint32,
	startSeq uint32,
	startCwnd uint32,
) {
	c.sockid = socketid
	c.pktSize = pktsz
	c.lastAck = 0
	c.ipc = send
	// Pseudo code doesn't specify how to intialize these
	c.lastAck = startSeq
	c.initCwnd = float64(10)
	c.cwnd = float64(startCwnd)
	c.ssthresh = (0x7fffffff / float64(pktsz))
	c.lastDrop = time.Now()
	c.rtt = time.Duration(0)
	// not sure about what this value should be
	c.cwnd_cnt = 0

	c.tcp_friendliness = true
	c.BETA = 0.2
	c.fast_convergence = true
	c.C = 0.4
	c.cubic_reset()

	pattern, err := pattern.
		NewPattern().
		Cwnd(uint32(c.cwnd * float64(c.pktSize))).
		WaitRtts(0.1).
		Report().
		Compile()
	if err != nil {
		log.WithFields(log.Fields{
			"err":      err,
			"cwndPkts": c.cwnd,
		}).Info("make cwnd msg failed")
		return
	}

	c.sendPattern(pattern)
}

func (c *Cubic) cubic_reset() {
	c.Wlast_max = 0
	c.epoch_start = 0
	c.origin_point = 0
	c.dMin = 0
	c.Wtcp = 0
	c.K = 0
	c.ack_cnt = 0
}

func (c *Cubic) GotMeasurement(m ccpFlow.Measurement) {
	// Ignore out of order netlink messages
	// Happens sometimes when the reporting interval is small
	// If within 10 packets, assume no integer overflow
	if m.Ack < c.lastAck && m.Ack > c.lastAck-c.pktSize*10 {
		return
	}

	// Handle integer overflow / sequence wraparound
	var newBytesAcked uint64
	if m.Ack < c.lastAck {
		newBytesAcked = uint64(math.MaxUint32) + uint64(m.Ack) - uint64(c.lastAck)
	} else {
		newBytesAcked = uint64(m.Ack) - uint64(c.lastAck)
	}

	c.rtt = m.Rtt
	RTT := float64(c.rtt.Seconds())
	no_of_acks := float64(newBytesAcked) / float64(c.pktSize)

	if c.cwnd <= c.ssthresh {
		if c.cwnd+no_of_acks < c.ssthresh {
			c.cwnd += no_of_acks
			no_of_acks = 0
		} else {
			no_of_acks -= (c.ssthresh - c.cwnd)
			c.cwnd = c.ssthresh
		}
	}

	for i := uint32(0); i < uint32(no_of_acks); i++ {
		if c.dMin <= 0 || RTT < c.dMin {
			c.dMin = RTT
		}

		c.cubic_update()
		if c.cwnd_cnt > c.cnt {
			c.cwnd = c.cwnd + 1
			c.cwnd_cnt = 0
		} else {
			c.cwnd_cnt = c.cwnd_cnt + 1
		}
	}

	// notify increased cwnd
	pattern, err := pattern.
		NewPattern().
		Cwnd(uint32(c.cwnd * float64(c.pktSize))).
		WaitRtts(0.5).
		Report().
		Compile()
	if err != nil {
		log.WithFields(log.Fields{
			"err":      err,
			"cwndPkts": c.cwnd,
		}).Info("make cwnd msg failed")
		return
	}

	c.sendPattern(pattern)

	log.WithFields(log.Fields{
		"gotAck":            m.Ack,
		"currLastAck":       c.lastAck,
		"newlyAckedPackets": float64(newBytesAcked) / float64(c.pktSize),
		"currCwndPkts":      c.cwnd,
	}).Info("[cubic] got ack")

	c.lastAck = m.Ack
	return
}

func (c *Cubic) Drop(ev ccpFlow.DropEvent) {
	if time.Since(c.lastDrop) <= c.rtt {
		return
	}

	c.lastDrop = time.Now()

	switch ev {
	case ccpFlow.DupAck:
		c.epoch_start = 0
		if c.cwnd < c.Wlast_max && c.fast_convergence {
			c.Wlast_max = c.cwnd * ((2 - c.BETA) / 2)
		} else {
			c.Wlast_max = c.cwnd
		}

		c.cwnd = c.cwnd * (1 - c.BETA)
		c.ssthresh = c.cwnd
	case ccpFlow.Timeout:
		c.ssthresh = c.cwnd / 2
		c.cwnd = c.initCwnd
		c.cubic_reset()
	default:
		log.WithFields(log.Fields{
			"event": ev,
		}).Warn("[cubic] unknown drop event type")
		return
	}

	pattern, err := pattern.
		NewPattern().
		Cwnd(uint32(c.cwnd * float64(c.pktSize))).
		WaitRtts(0.1).
		Report().
		Compile()
	if err != nil {
		log.WithFields(log.Fields{
			"err":      err,
			"cwndPkts": c.cwnd,
		}).Info("make cwnd msg failed")
		return
	}

	c.sendPattern(pattern)
	log.WithFields(log.Fields{
		"currCwndPkts": c.cwnd,
		"event":        ev,
	}).Info("[cubic] drop")
}

func (c *Cubic) cubic_update() {
	c.ack_cnt = c.ack_cnt + 1
	if c.epoch_start <= 0 {
		c.epoch_start = float64(time.Now().UnixNano() / 1e9)
		if c.cwnd < c.Wlast_max {
			c.K = math.Pow(math.Max(0.0, ((c.Wlast_max-c.cwnd)/c.C)), 1.0/3.0)
			c.origin_point = c.Wlast_max
		} else {
			c.K = 0
			c.origin_point = c.cwnd
		}

		c.ack_cnt = 1
		c.Wtcp = c.cwnd
	}

	t := float64(time.Now().UnixNano()/1e9) + c.dMin - c.epoch_start
	target := c.origin_point + c.C*((t-c.K)*(t-c.K)*(t-c.K))
	if target > c.cwnd {
		c.cnt = c.cwnd / (target - c.cwnd)
	} else {
		c.cnt = 100 * c.cwnd
	}

	if c.tcp_friendliness {
		c.cubic_tcp_friendliness()
	}
}

func (c *Cubic) cubic_tcp_friendliness() {
	c.Wtcp = c.Wtcp + (((3 * c.BETA) / (2 - c.BETA)) * (c.ack_cnt / c.cwnd))
	c.ack_cnt = 0
	if c.Wtcp > c.cwnd {
		max_cnt := c.cwnd / (c.Wtcp - c.cwnd)
		if c.cnt > max_cnt {
			c.cnt = max_cnt
		}
	}
}

func (c *Cubic) sendPattern(pattern *pattern.Pattern) {
	err := c.ipc.SendPatternMsg(c.sockid, pattern)
	if err != nil {
		log.WithFields(log.Fields{
			"cwndPkts": c.cwnd,
			"name":     c.sockid,
		}).Warn(err)
	}
}

func Init() {
	ccpFlow.Register("cubic", func() ccpFlow.Flow {
		return &Cubic{}
	})
}
