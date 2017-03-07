package simbackend

import (
	"simbackend/ipc"
	"simbackend/mmap"
)

func (sock *Sock) setupIpc() error {
	ipc, err := mmap.Setup()
	if err != nil {
		return err
	}

	sock.ipc = ipc

	// start listening for cwnd changes
	cwndChanges, err := sock.ipc.Listen()
	if err != nil {
		return err
	}

	go func(ch chan uint32) {
		for cwnd := range ch {
			sock.cwnd = cwnd
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
	err := out.Send(0, ack)
	if err != nil {
		return
	}
}
