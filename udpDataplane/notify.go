package udpDataplane

import (
	"ccp/ipc"
)

func (sock *Sock) setupIpc(sockid uint32) error {
	ipcL, err := ipc.Setup(sockid)
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
			sock.cwnd = cwnd.Cwnd
		}
	}(cwndChanges)

	return nil
}

func (sock *Sock) notifyAcks() {
	if sock.cumAck-sock.notifiedAckNo > sock.ackNotifyThresh {
		// notify control plane of new acks
		sock.notifiedAckNo = sock.cumAck
		go writeAckMsg(sock.ipc, sock.notifiedAckNo)
	}
}

func writeAckMsg(out ipc.Ipc, ack uint32) {
	err := out.SendAckMsg(0, ack)
	if err != nil {
		return
	}
}
