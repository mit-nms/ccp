package compound

import (
	"math"
	"time"

	"ccp/ccpFlow"
	"ccp/ccpFlow/pattern"
	"ccp/ipc"

	log "github.com/sirupsen/logrus"
)

//Specification from https://tools.ietf.org/html/draft-sridharan-tcpm-ctcp-02#section-3 and http://www.dcs.gla.ac.uk/~lewis/CTCP.pdf

// implement ccpFlow.Flow interface
type Compound struct {
	pktSize  uint32
	initCwnd float32

	wnd     float32
	cwnd    float32
	dwnd    float32
	lastAck uint32

	sockid     uint32
	ipc        ipc.SendOnly
	baseRTT    float32
	alpha      float32
	beta       float32
	k          float64
	eta        float32
	gamma      float32
	gamma_low  float32
	gamma_high float32
	diff_reno  float32
	lastDrop   time.Time
}

func (c *Compound) Name() string {
	return "compound"
}

func (c *Compound) Create(
	socketid uint32,
	send ipc.SendOnly,
	pktsz uint32,
	startSeq uint32,
	startCwnd uint32,
) {
	c.sockid = socketid
	c.ipc = send
	c.pktSize = pktsz
	c.initCwnd = float32(pktsz * 10)
	c.wnd = float32(pktsz * startCwnd)
	c.cwnd = float32(pktsz * startCwnd)
	c.dwnd = 0
	if startSeq == 0 {
		c.lastAck = startSeq
	} else {
		c.lastAck = startSeq - 1
	}
	c.baseRTT = 0
	c.alpha = 0.125
	c.beta = 0.5
	c.k = 0.8
	c.eta = 1
	c.gamma = 30
	c.gamma_low = 5
	c.gamma_high = 30
	c.diff_reno = -1
	c.lastDrop = time.Now()
	c.newPattern()
}

func (c *Compound) GotMeasurement(m ccpFlow.Measurement) {
	if m.Ack < c.lastAck {
		return
	}

	RTT := float32(m.Rtt.Seconds())
	if c.baseRTT <= 0 || RTT < c.baseRTT {
		c.baseRTT = RTT
	}

	newBytesAcked := float32(m.Ack - c.lastAck)
	// increase cwnd by 1 / cwnd per packet
	c.cwnd += float32(c.pktSize) * (newBytesAcked / c.wnd)

	//dwnd update
	expected := c.wnd / c.baseRTT
	actual := c.wnd / RTT
	diff := (expected - actual) * c.baseRTT
	c.diff_reno = diff
	increment := float32(0.0)
	if diff < c.gamma*float32(c.pktSize) {
		increment = float32(c.pktSize) * ((c.alpha * float32(math.Pow(float64(c.wnd)/float64(c.pktSize), c.k))) - 1)
		if increment < 0 {
			increment = 0
		}
	} else {
		increment = -c.eta * diff
	}
	c.dwnd += increment * (newBytesAcked / c.wnd)
	if c.dwnd < 0 {
		c.dwnd = 0
	}
	//wnd update
	c.wnd = c.cwnd + c.dwnd
	// notify increased cwnd

	c.newPattern()

	log.WithFields(log.Fields{
		"gotAck":      m.Ack,
		"currCwnd":    c.wnd,
		"currLastAck": c.lastAck,
		"newlyAcked":  newBytesAcked,
	}).Info("[compound] got ack")


	c.lastAck = m.Ack

	return
}

func (c *Compound) Drop(ev ccpFlow.DropEvent) {
	//if float32(time.Since(c.lastDrop).Seconds()) <= c.baseRTT {
    //    return
    //}
    //c.lastDrop = time.Now()
	oldCwnd := c.wnd
	switch ev {
	case ccpFlow.DupAck:
		c.cwnd /= 2
		if c.cwnd < c.initCwnd {
			c.cwnd = c.initCwnd
		}
		c.dwnd = c.wnd*(1-c.beta) - c.cwnd
		if c.dwnd < 0 {
			c.dwnd = 0
		}
		c.wnd = c.cwnd + c.dwnd
		//gamma auto tuning
		if c.diff_reno >= 0 {
			g_sample := 0.75 * (c.diff_reno / float32(c.pktSize))
			lambda := float32(0.8) //couldn't find how to set this
			c.gamma = lambda*c.gamma + (1-lambda)*g_sample
			if c.gamma < c.gamma_low {
				c.gamma = c.gamma_low
			} else if c.gamma > c.gamma_high {
				c.gamma = c.gamma_high
			}
			c.diff_reno = -1
		}
	case ccpFlow.Timeout:
		c.wnd = c.initCwnd
		c.cwnd = c.initCwnd
		c.dwnd = 0
		c.gamma = 30
	default:
		log.WithFields(log.Fields{
			"event": ev,
		}).Warn("[reno] unknown drop event type")
		return
	}

	log.WithFields(log.Fields{
		"oldCwnd":  oldCwnd,
		"currCwnd": c.wnd,
		"event":    ev,
	}).Info("[compound] drop")

	c.newPattern()
}

func (c *Compound) newPattern() {
	staticPattern, err := pattern.
		NewPattern().
		Cwnd(uint32(c.wnd)).
		Wait(time.Duration(10) * time.Millisecond).
		Report().
		Compile()
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"cwnd": c.wnd,
		}).Info("make cwnd msg failed")
		return
	}

	err = c.ipc.SendPatternMsg(c.sockid, staticPattern)
	if err != nil {
		log.WithFields(log.Fields{"cwnd": c.wnd, "name": c.sockid}).Warn(err)
	}
}

func Init() {
	ccpFlow.Register("compound", func() ccpFlow.Flow {
		return &Compound{}
	})
}
