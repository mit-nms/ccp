package simbackend

import (
	"simbackend/mmap"

	capnpMsg "simbackend/capnpMsg"
	"zombiezen.com/go/capnproto2"
)

func (sock *Sock) notifyAcks() {
	if sock.cumAck-sock.notifiedAckNo > sock.ackNotifyThresh {
		// notify control plane of new acks
		sock.notifiedAckNo = sock.cumAck

		// Cap'n Proto! Message passing interface to mmap file
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			return
		}

		notifyAckMsg, err := capnpMsg.NewNotifyAckMsg(seg)
		if err != nil {
			return
		}

		notifyAckMsg.SetSocketId(4)
		notifyAckMsg.SetAckNo(sock.notifiedAckNo)

		mm := mmap.Mmap("/tmp/ccp-in")

	}
}
