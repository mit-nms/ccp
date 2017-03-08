package udpDataplane

import (
	"ccp/ipc"
	"ccp/mmap"
)

func (sock *Sock) setupIpc() error {
	ipcL, err := mmap.Setup()
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

func writeAckMsg(out ipc.IpcLayer, ack uint32) {
	err := out.SendAckMsg(0, ack)
	if err != nil {
		return
	}
}
