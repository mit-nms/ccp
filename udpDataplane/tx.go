package udpDataplane

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.mit.edu/hari/nimbus-cc/packetops"
)

func (sock *Sock) nextPacket() (*Packet, error) {
	sock.mux.Lock()
	defer sock.mux.Unlock()

	seq := sock.inFlight.getNextPkt(sock.nextSeqNo)
	if seq >= uint32(sock.writeBufPos) {
		//nothing more to write
		return nil, fmt.Errorf("nothing more to write")
	}

	var payl []byte
	if packetEnd := seq + PACKET_SIZE; packetEnd <= uint32(sock.writeBufPos) {
		payl = sock.writeBuf[seq:packetEnd]
	} else {
		payl = sock.writeBuf[seq:sock.writeBufPos]
	}

	ackNo, err := sock.rcvWindow.cumAck(sock.lastAck)
	if err != nil {
		ackNo = sock.lastAck
		log.WithFields(log.Fields{"name": sock.name, "side": "rcvWindow", "ackNo": ackNo}).Debug(err)
	}

	sock.lastAck = ackNo

	pkt := &Packet{
		SeqNo:   seq,
		AckNo:   sock.lastAck,
		Flag:    ACK,
		Length:  uint16(len(payl)),
		Sack:    sock.rcvWindow.getSack(sock.lastAck),
		Payload: payl,
	}

	if seq == sock.nextSeqNo {
		sock.inFlight.addPkt(time.Now(), pkt)
		sock.nextSeqNo += uint32(pkt.Length)
	}
	return pkt, nil
}

func (sock *Sock) nextAck() (*Packet, error) {
	sock.mux.Lock()
	defer sock.mux.Unlock()

	ackNo, err := sock.rcvWindow.cumAck(sock.lastAck)
	if err != nil {
		ackNo = sock.lastAck
		log.WithFields(log.Fields{"name": sock.name, "side": "rcvWindow", "ackNo": ackNo}).Debug(err)
	}

	sock.lastAck = ackNo

	return &Packet{
		SeqNo:   sock.nextSeqNo,
		AckNo:   sock.lastAck,
		Flag:    ACK,
		Length:  0,
		Sack:    sock.rcvWindow.getSack(sock.lastAck),
		Payload: []byte{},
	}, nil
}

func (sock *Sock) tx() {
	for {
		select {
		case <-sock.closed:
			log.WithFields(log.Fields{"where": "tx", "name": sock.name}).Debug("closed, exiting")
			return
		case <-sock.shouldTx:
			// got new ack
			// or got new data and want to send ack
			log.WithFields(log.Fields{
				"name": sock.name,
			}).Debug("tx")
		case <-time.After(time.Duration(3) * time.Second):
			// timeout, assume entire window lost
			if len(sock.inFlight.order) > 0 {
				log.WithFields(log.Fields{
					"name":          sock.name,
					"sock.inFlight": sock.inFlight.order,
				}).Debug("timeout!")
				firstUnacked, err := sock.inFlight.start()
				if err != nil {
					break
				}

				sock.dupAckCnt = 0
				sock.inFlight.timeout()
				select {
				case sock.notifyDrops <- notifyDrop{ev: "timeout", lastAck: firstUnacked}:
				default:
				}
			}
		}

		sock.doTx()
	}
}

func (sock *Sock) doTx() {
	sent := false
	sock.mux.Lock()
	cwnd := sock.cwnd
	sock.mux.Unlock()
	for sock.inFlight.size() < cwnd {
		pkt, err := sock.nextPacket()
		if err != nil {
			log.WithFields(log.Fields{
				"name": sock.name,
			}).Debug(err)
			break
		}

		sent = true

		packetops.SendPacket(sock.conn, pkt, 0)

		log.WithFields(log.Fields{
			"name":          sock.name,
			"bytesInFlight": sock.inFlight.size(),
			"seqNo":         pkt.SeqNo,
			"ackNo":         pkt.AckNo,
			"length":        pkt.Length,
			"inFlight":      sock.inFlight.getOrder(),
			"cwnd":          cwnd,
		}).Debug("sent packet")
	}

	// send an ACK
	if !sent {
		pkt, err := sock.nextAck()
		if err != nil {
			return
		}

		packetops.SendPacket(sock.conn, pkt, 0)

		log.WithFields(log.Fields{
			"name":  sock.name,
			"ackNo": pkt.AckNo,
		}).Debug("sent ack")
	}
}
