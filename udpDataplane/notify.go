package udpDataplane

import (
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

func (sock *Sock) notifyAcks() {
	if sock.lastAckedSeqNo-sock.notifiedAckNo > sock.ackNotifyThresh {
		// notify control plane of new acks
		sock.notifiedAckNo = sock.lastAckedSeqNo
		go writeAckMsg(sock.name, sock.ipc, sock.notifiedAckNo)
	}

	if sock.conn == nil {
		log.WithFields(log.Fields{"ack": sock.lastAckedSeqNo, "name": sock.name}).Debug("closed")
		return
	}

	select {
	case sock.ackedData <- sock.lastAckedSeqNo:
		log.WithFields(log.Fields{"ack": sock.lastAckedSeqNo, "name": sock.name}).Debug("send ack notification to app")
	case <-sock.closed:
		close(sock.ackedData)
	default:
		log.WithFields(log.Fields{"ack": sock.lastAckedSeqNo, "name": sock.name}).Debug("not blocking on ack notif")
	}
}

func writeAckMsg(name string, out *ipc.Ipc, ack uint32) {
	err := out.SendAckMsg(0, ack)
	if err != nil {
		log.WithFields(log.Fields{"ack": ack, "name": name}).Warn(err)
		return
	}

	log.WithFields(log.Fields{"ack": ack, "name": name}).Debug("send ack notification to ccp")
}
