package simbackend

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.mit.edu/hari/nimbus-cc/packetops"
)

func (sock *Sock) rx() {
	rcvd := &Packet{}
	for {
		err := sock.doRx(rcvd)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func (sock *Sock) doRx(rcvd *Packet) error {
	_, err := packetops.RecvPacket(sock.conn, rcvd)
	if err != nil {
		return err
	}

	if rcvd.Flag == FIN {
		sock.Close()
		return nil
	} else if rcvd.Flag != ACK {
		var flag string
		switch rcvd.Flag {
		case SYN:
			flag = "SYN"
		case SYNACK:
			flag = "SYNACK"
		default:
			flag = "unknown"
		}

		log.WithFields(log.Fields{
			"flag": flag,
			"pkt":  rcvd,
		}).Panic("connection in unknown state")
	}

	rcvd.Payload = rcvd.Payload[:rcvd.Length]

	sock.mux.Lock()
	sock.handleAck(rcvd)
	sock.handleData(rcvd)
	sock.mux.Unlock()

	log.WithFields(log.Fields{
		"sock.name":   sock.name,
		"pkt.seqNo":   rcvd.SeqNo,
		"pkt.ackNo":   rcvd.AckNo,
		"sock.cumAck": sock.cumAck,
		"sock.ackNo":  sock.ackNo,
	}).Info("received packet")

	return nil
}

// process ack
// go back N
// require exactly in order delivery
func (sock *Sock) handleAck(rcvd *Packet) {
	if rcvd.AckNo > sock.cumAck {
		sock.inFlight -= (rcvd.AckNo - sock.cumAck)
		sock.cumAck = rcvd.AckNo
		sock.shouldTx <- struct{}{}

		log.WithFields(log.Fields{
			"name":             sock.name,
			"sock.cumAck":      sock.cumAck,
			"sock.writeBufPos": sock.writeBufPos,
			"rcvd":             rcvd,
		}).Info("new ack")

		sock.notifyAcks()
	}
}

// process received payload
// go back N
// require exactly in order delivery
func (sock *Sock) handleData(rcvd *Packet) {
	if rcvd.SeqNo == sock.ackNo && rcvd.Length > 0 {
		// new data!
		log.WithFields(log.Fields{
			"name":        sock.name,
			"rcvd length": rcvd.Length,
			"rcvd":        rcvd,
		}).Info("new data")

		sock.ackNo += uint32(rcvd.Length)
		copy(sock.readBuf[rcvd.SeqNo:], rcvd.Payload[:rcvd.Length])
		sock.shouldPass <- struct{}{}
		sock.shouldTx <- struct{}{}
	}
}
