package udpDataplane

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.mit.edu/hari/nimbus-cc/packetops"
)

func (sock *Sock) nextPacket() (*Packet, error) {
	if sock.seqNo >= uint32(sock.writeBufPos) {
		//nothing more to write
		return nil, fmt.Errorf("nothing more to write")
	}

	var payl []byte
	if packetEnd := sock.seqNo + PACKET_SIZE; packetEnd <= uint32(sock.writeBufPos) {
		payl = sock.writeBuf[sock.seqNo:packetEnd]
	} else {
		payl = sock.writeBuf[sock.seqNo:sock.writeBufPos]
	}

	return &Packet{
		SeqNo:   sock.seqNo,
		AckNo:   sock.ackNo,
		Flag:    ACK,
		Length:  uint16(len(payl)),
		Payload: payl,
	}, nil
}

func (sock *Sock) nextAck() (*Packet, error) {
	if sock.ackNo <= sock.lastAckSent {
		return nil, fmt.Errorf("already acked")
	}

	sock.lastAckSent = sock.ackNo

	return &Packet{
		SeqNo:   sock.seqNo,
		AckNo:   sock.ackNo,
		Flag:    ACK,
		Length:  0,
		Payload: []byte{},
	}, nil
}

func (sock *Sock) tx() {
	for {
		select {
		case <-sock.shouldTx:
			// got new ack
			// or got new data and want to send ack
			log.WithFields(log.Fields{
				"name": sock.name,
			}).Info("tx")
		case <-time.After(time.Duration(3) * time.Second):
			// timeout, assume entire window lost
			// go back N
			sock.seqNo = sock.cumAck
			sock.inFlight = 0
			log.Info("timeout!")
		}

		if sock.checkClosed() {
			return
		}

		sock.doTx()
	}
}

func (sock *Sock) doTx() {
	sent := false
	for sock.inFlight < sock.cwnd {
		sock.mux.Lock()
		pkt, err := sock.nextPacket()
		sock.mux.Unlock()

		if err != nil {
			log.WithFields(log.Fields{
				"name": sock.name,
			}).Info(err)
			break
		}

		sent = true

		packetops.SendPacket(sock.conn, pkt, 0)
		sock.inFlight += uint32(pkt.Length)
		sock.seqNo += uint32(pkt.Length)

		log.WithFields(log.Fields{
			"name":     sock.name,
			"seqNo":    pkt.SeqNo,
			"ackNo":    pkt.AckNo,
			"payload":  string(pkt.Payload),
			"pkt":      pkt,
			"inFlight": sock.inFlight,
			"cwnd":     sock.cwnd,
			"cumAck":   sock.cumAck,
		}).Info("sent packet")
	}

	// send an ACK
	if !sent {
		sock.mux.Lock()
		pkt, err := sock.nextAck()
		sock.mux.Unlock()
		if err != nil {
			return
		}

		packetops.SendPacket(sock.conn, pkt, 0)

		log.WithFields(log.Fields{
			"name":  sock.name,
			"ackNo": pkt.AckNo,
			"pkt":   pkt,
		}).Info("sent ack")
	}
}
