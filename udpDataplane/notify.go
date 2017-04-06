package udpDataplane

import (
	"time"

	"ccp/ccpFlow"
	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
)

func (sock *Sock) setupIpc() error {
	ipcL, err := ipc.SetupCli(sock.port)
	if err != nil {
		return err
	}

	sock.ipc = ipcL

	// start listening for cwnd changes
	cwndChanges, err := sock.ipc.ListenCwndMsg()
	if err != nil {
		return err
	}

	go func(ch chan ipc.CwndMsg) {
		for cwnd := range ch {
			log.WithFields(log.Fields{
				"newCwnd": cwnd.Cwnd,
				"cwnd":    sock.cwnd,
			}).Info("update cwnd")
			sock.mux.Lock()
			sock.cwnd = cwnd.Cwnd
			sock.mux.Unlock()
			sock.shouldTx <- struct{}{}
		}
	}(cwndChanges)
	go sock.doNotify()

	sock.ipc.SendCreateMsg(sock.port, "reno")
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
	err := out.SendAckMsg(id, ack, rtt)
	if err != nil {
		log.WithFields(log.Fields{"ack": ack, "name": name, "id": id, "where": "notify.writeAckMsg"}).Warn(err)
		return
	}
}

func writeDropMsg(name string, id uint32, out *ipc.Ipc, event string) {
	switch event {
	case "timeout":
		event = string(ccpFlow.Complete)
	case "3xdupack":
		event = string(ccpFlow.Isolated)
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
