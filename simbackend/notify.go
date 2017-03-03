package simbackend

import (
	"simbackend/mmap"

	capnpMsg "simbackend/capnpMsg"
	"zombiezen.com/go/capnproto2"
)

func (sock *Sock) setupMmap() error {
	mm, err := mmap.Mmap("/tmp/ccp-in")
	if err != nil {
		return err
	}

	sock.mmOut = mm

	mm, err = mmap.Mmap("/tmp/ccp-out")
	if err != nil {
		return err
	}

	sock.mmIn = mm

	return nil
}

func (sock *Sock) teardownMmap() error {
	sock.mmIn.Close()
	sock.mmOut.Close()

	return nil
}

func (sock *Sock) notifyAcks() {
	if sock.cumAck-sock.notifiedAckNo > sock.ackNotifyThresh {
		// notify control plane of new acks
		sock.notifiedAckNo = sock.cumAck
		go writeAckMsg(sock.mmOut, sock.notifiedAckNo)
	}
}

func writeAckMsg(out mmap.MM, ack uint32) {
	// Cap'n Proto! Message passing interface to mmap file
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return
	}

	notifyAckMsg, err := capnpMsg.NewRootNotifyAckMsg(seg)
	if err != nil {
		return
	}

	notifyAckMsg.SetSocketId(4)
	notifyAckMsg.SetAckNo(ack)

	err = out.Enc.Encode(msg)
	if err != nil {
		return
	}
}
