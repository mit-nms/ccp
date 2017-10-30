package udpDataplane

import (
	"time"

	"ccp/ccpFlow"
	"ccp/ccpFlow/pattern"
	"ccp/ipc"

	log "github.com/sirupsen/logrus"
)

type notifyAck struct {
	ack uint32
	rtt time.Duration
}

type notifyDrop struct {
	lastAck uint32
	ev      string
}

func (sock *Sock) ipcListen(patternCh chan ipc.PatternMsg, measureMsgs chan notifyAck) {
	patternChanged := make(chan *pattern.Pattern)
	go func() {
		stopPattern := make(chan interface{})
		for newPattern := range patternChanged {
			select {
			case stopPattern <- struct{}{}:
			default:
			}

			go sock.doPattern(newPattern, measureMsgs, stopPattern)
		}
	}()

	for {
		select {
		case pmsg := <-patternCh:
			patternChanged <- pmsg.Pattern()
		case <-sock.closed:
			close(patternChanged)
			return
		}
	}
}

func (sock *Sock) setupIpc() error {
	ipcL, err := ipc.SetupCli(sock.port)
	if err != nil {
		return err
	}

	sock.ipc = ipcL

	// start listening for pattern specifications
	patternSet, err := sock.ipc.ListenPatternMsg()
	if err != nil {
		return err
	}

	measureMsgWaiter := make(chan notifyAck)
	go sock.doNotify(measureMsgWaiter)
	go sock.ipcListen(patternSet, measureMsgWaiter)

	sock.ipc.SendCreateMsg(sock.port, 0, "reno")
	return nil
}

func (sock *Sock) doNotify(measureMsg chan notifyAck) {
	totAck := uint32(0)
	droppedPktNo := uint32(0)
	timeout := time.NewTimer(time.Second)
	for {
		select {
		case notifAck := <-sock.notifyAcks:
			if notifAck.ack > totAck {
				totAck = notifAck.ack
			} else {
				continue
			}

			log.WithFields(log.Fields{
				"name":  sock.name,
				"acked": totAck,
				"rtt":   notifAck.rtt,
			}).Info("notifyAcks")

			select {
			case measureMsg <- notifAck:
			default:
			}

			select {
			case sock.ackedData <- totAck:
			default:
			}
		case dropEv := <-sock.notifyDrops:
			if dropEv.lastAck > droppedPktNo {
				log.WithFields(log.Fields{
					"name":  sock.name,
					"event": dropEv,
				}).Info("notifyDrops")
				writeDropMsg(sock.name, sock.port, sock.ipc, dropEv.ev)
				droppedPktNo = dropEv.lastAck
			}
		case <-timeout.C:
			timeout.Reset(time.Second)
		case <-sock.closed:
			log.WithFields(log.Fields{"where": "doNotify", "name": sock.name}).Debug("closed, exiting")
			close(sock.ackedData)
			return
		}
		if !timeout.Stop() {
			<-timeout.C
		}
		timeout.Reset(time.Second)
	}
}

func (sock *Sock) doNonWaitPatternEvent(ev pattern.PatternEvent, currRtt time.Duration) {
	var cwnd uint32
	switch ev.Type {
	case pattern.REPORT:
		fallthrough
	case pattern.WAITREL:
		fallthrough
	case pattern.WAITABS:
		return

	case pattern.SETRATEABS:
		// set cwnd = rate * rtt
		cwnd = uint32(float64(ev.Rate) * currRtt.Seconds())

	case pattern.SETCWNDABS:
		cwnd = ev.Cwnd

	case pattern.SETRATEREL:
		cwnd = uint32(float64(sock.cwnd) * float64(ev.Factor))
	}

	log.WithFields(log.Fields{
		"cwnd": cwnd,
	}).Info("set cwnd")
	sock.mux.Lock()
	sock.cwnd = cwnd
	sock.mux.Unlock()
}

func (sock *Sock) doPattern(p *pattern.Pattern, measureMsgs chan notifyAck, stopPattern chan interface{}) {
	// infinitely loop through events in the sequence
	var currRtt time.Duration
	for {
		for _, ev := range p.Sequence {
			wait := time.Duration(0)
			switch ev.Type {
			case pattern.WAITABS:
				wait = ev.Duration
			case pattern.WAITREL:
				wait = time.Duration(currRtt.Seconds()*float64(ev.Factor)*1e6) * time.Microsecond

			case pattern.REPORT:
				select {
				case meas := <-measureMsgs:
					currRtt = meas.rtt
					writeMeasureMsg(sock.name, sock.port, sock.ipc, meas.ack, meas.rtt)
					continue
				case <-stopPattern:
					return
				}

			default:
				sock.doNonWaitPatternEvent(ev, currRtt)
				continue
			}

			select {
			case <-time.After(wait):
				continue
			case <-stopPattern:
				return
			}
		}
	}
}

func writeMeasureMsg(
	name string,
	id uint32,
	out *ipc.Ipc,
	ack uint32,
	rtt time.Duration,
) {
	err := out.SendMeasureMsg(id, ack, rtt, 0, 0, 0)
	if err != nil {
		log.WithFields(log.Fields{"ack": ack, "name": name, "id": id, "where": "notify.writeMeasureMsg"}).Warn(err)
		return
	}
}

func writeDropMsg(name string, id uint32, out *ipc.Ipc, event string) {
	switch event {
	case "timeout":
		event = string(ccpFlow.Timeout)
	case "3xdupack":
		event = string(ccpFlow.DupAck)
	default:
		log.WithFields(log.Fields{
			"event": event,
			"id":    id,
			"name":  name,
		}).Panic("unknown event")
	}
	err := out.SendDropMsg(id, event)
	if err != nil {
		log.WithFields(log.Fields{"event": event, "name": name, "id": id, "where": "notify.writeDropMsg"}).Warn(err)
		return
	}
}
