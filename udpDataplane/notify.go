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
		}
	}(cwndChanges)

	return nil
}

func (sock *Sock) notifyAcks() {
	if sock.cumAck-sock.notifiedAckNo > sock.ackNotifyThresh {
		// notify control plane of new acks
		sock.notifiedAckNo = sock.cumAck
		go writeAckMsg(sock.name, sock.ipc, sock.notifiedAckNo)
	}

	select {
	case sock.ackedData <- sock.cumAck:
	default:
	}
}

func writeAckMsg(name string, out *ipc.Ipc, ack uint32) {
	err := out.SendAckMsg(0, ack)
	if err != nil {
		log.WithFields(log.Fields{"ack": ack, "name": name}).Warn(err)
		return
	}

	log.WithFields(log.Fields{"ack": ack, "name": name}).Info("send ack notification to ccp")
}
