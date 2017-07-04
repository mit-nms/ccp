package udpDataplane

import (
	"time"

	"ccp/ccpFlow"
	"ccp/ccpFlow/pattern"
	"ccp/ipc"

	log "github.com/sirupsen/logrus"
)

// TODO correct RTTs
func (sock *Sock) doNonWaitPatternEvent(ev pattern.PatternEvent) {
	switch ev.Type {
	case pattern.WAITREL:
		fallthrough
	case pattern.WAITABS:
		return

	case pattern.REPORT:
		// make it work together with notifyAck

	case pattern.SETCWNDABS:
		sock.mux.Lock()
		sock.cwnd = ev.Value
		sock.mux.Unlock()

	case pattern.SETRATEABS:
		// set cwnd = rate * rtt
		cwnd := float64(ev.Value) * (time.Duration(20) * time.Millisecond).Seconds()
		sock.mux.Lock()
		sock.cwnd = uint32(cwnd)
		sock.mux.Unlock()

	case pattern.SETRATEREL:
		// current rate?
		// cwnd (bytes) / rtt
		currRate := float64(sock.cwnd) / (time.Duration(20) * time.Millisecond).Seconds()
		wantedRate := currRate * float64(ev.Value)
		cwnd := wantedRate * (time.Duration(20) * time.Millisecond).Seconds()
		sock.mux.Lock()
		sock.cwnd = uint32(cwnd)
		sock.mux.Unlock()
	}
}

func (sock *Sock) doPattern(p *pattern.Pattern, stopPattern chan interface{}) {
	// infinitely loop through events in the sequence
	for {
		for _, ev := range p.Sequence {
			wait := time.Duration(0)
			switch ev.Type {
			case pattern.WAITABS:
				wait = ev.Duration
			case pattern.WAITREL:
				wait = time.Duration(20*ev.Value) * time.Millisecond
			default:
				sock.doNonWaitPatternEvent(ev)
			}

			if wait != time.Duration(0) {
				select {
				case <-time.After(wait):
					continue
				case <-stopPattern:
					return
				}
			}
		}
	}
}

func (sock *Sock) ipcListen(setCh chan ipc.SetMsg, patternCh chan ipc.PatternMsg) {
	patternChanged := make(chan *pattern.Pattern)
	go func() {
		stopPattern := make(chan interface{})
		for newPattern := range patternChanged {
			select {
			case stopPattern <- struct{}{}:
			default:
			}

			go sock.doPattern(newPattern, stopPattern)
		}
	}()
	for {
		select {
		case smsg := <-setCh:
			log.WithFields(log.Fields{
				"newSet":   smsg.Set(),
				"setMode":  smsg.Mode(),
				"currCwnd": sock.cwnd,
			}).Info("update cwnd")
			switch smsg.Mode() {
			case "cwnd":
				sock.mux.Lock()
				sock.cwnd = smsg.Set()
				sock.mux.Unlock()
			case "rate":
				// set cwnd = rate * rtt
				// TODO keep RTT estimate around somewhere
				cwnd := float64(smsg.Set()) * (time.Duration(20) * time.Millisecond).Seconds()
				sock.mux.Lock()
				sock.cwnd = uint32(cwnd)
				sock.mux.Unlock()
			}
			sock.shouldTx <- struct{}{}
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

	// start listening for cwnd changes
	cwndChanges, err := sock.ipc.ListenSetMsg()
	if err != nil {
		return err
	}

	patternSet, err := sock.ipc.ListenPatternMsg()
	if err != nil {
		return err
	}

	go sock.ipcListen(cwndChanges, patternSet)
	go sock.doNotify()

	sock.ipc.SendCreateMsg(sock.port, 0, "reno")
	return nil
}

type notifyAck struct {
	ack uint32
	rtt time.Duration
}

type notifyDrop struct {
	lastAck uint32
	ev      string
}

func (sock *Sock) doNotify() {
	totAck := uint32(0)
	notifiedAckNo := uint32(0)
	droppedPktNo := uint32(0)
	srtt := time.Duration(0)
	timeout := time.NewTimer(time.Second)
	for {
		select {
		case notifAck := <-sock.notifyAcks:
			if notifAck.ack > totAck {
				totAck = notifAck.ack
			}

			log.WithFields(log.Fields{
				"name":          sock.name,
				"acked":         totAck,
				"rtt":           notifAck.rtt,
				"notifiedAckNo": notifiedAckNo,
			}).Info("notifyAcks")

			if srtt == time.Duration(0) {
				srtt = notifAck.rtt
			} else {
				srtt_ns := 0.875*float64(srtt.Nanoseconds()) + 0.125*float64(notifAck.rtt)
				srtt = time.Duration(int64(srtt_ns)) * time.Nanosecond
			}

			if totAck-notifiedAckNo > sock.ackNotifyThresh {
				// notify control plane of new acks
				notifiedAckNo = totAck
				writeAckMsg(sock.name, sock.port, sock.ipc, notifiedAckNo, srtt)
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

func writeAckMsg(
	name string,
	id uint32,
	out *ipc.Ipc,
	ack uint32,
	rtt time.Duration,
) {
	err := out.SendMeasureMsg(id, ack, rtt, 0, 0)
	if err != nil {
		log.WithFields(log.Fields{"ack": ack, "name": name, "id": id, "where": "notify.writeAckMsg"}).Warn(err)
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
