package udpDataplane

import (
	"time"

	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
)

func (sock *Sock) setupIpc(sockid uint32) error {
	ipcL, err := ipc.SetupCli(sockid)
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
			sock.mux.Lock()
			sock.cwnd = cwnd.Cwnd
			sock.mux.Unlock()
			sock.shouldTx <- struct{}{}
		}
	}(cwndChanges)

	sock.ipc.SendCreateMsg(sockid, "reno")
	return nil
}

func (sock *Sock) doNotifyAcks() {
	totAck := uint32(0)
	notifiedAckNo := uint32(0)
	timeout := time.NewTimer(time.Second)
	for {
		select {
		case ack := <-sock.notifyAcks:
			if ack > totAck {
				totAck = ack
			}

			log.WithFields(log.Fields{
				"name":          sock.name,
				"acked":         totAck,
				"notifiedAckNo": notifiedAckNo,
			}).Info("got send on notifyAcks")

			if !timeout.Stop() {
				<-timeout.C
			}
			timeout.Reset(time.Second)
		case <-timeout.C:
			timeout.Reset(time.Second)
		case <-sock.closed:
			log.WithFields(log.Fields{"where": "doNotifyAcks", "name": sock.name}).Debug("closed, exiting")
			close(sock.ackedData)
			return
		}

		if totAck-notifiedAckNo > sock.ackNotifyThresh {
			// notify control plane of new acks
			notifiedAckNo = totAck
			writeAckMsg(sock.name, sock.ipc, notifiedAckNo)
		}

		select {
		case sock.ackedData <- totAck:
			log.WithFields(log.Fields{"ack": totAck, "name": sock.name}).Info("send ack notification to app")
		default:
		}
	}
}

func writeAckMsg(name string, out *ipc.Ipc, ack uint32) {
	err := out.SendAckMsg(0, ack)
	if err != nil {
		log.WithFields(log.Fields{"ack": ack, "name": name, "where": "sending ack to ccp"}).Warn(err)
		return
	}

	log.WithFields(log.Fields{"ack": ack, "name": name}).Info("send ack notification to ccp")
}
