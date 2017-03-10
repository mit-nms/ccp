package udpDataplane

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.mit.edu/hari/nimbus-cc/packetops"
)

func (sock *Sock) rx() {
	rcvdPkts := make(chan *Packet)
	go func() {
		for {
			select {
			case <-sock.closed:
				return
			default:
			}

			rcvd := &Packet{}
			_, err := packetops.RecvPacket(sock.conn, rcvd)
			if err != nil {
				log.WithFields(log.Fields{"where": "rx"}).Warn(err)
				continue
			}

			rcvdPkts <- rcvd
		}
	}()

	for {
		select {
		case <-sock.closed:
			return
		case rcvd := <-rcvdPkts:
			err := sock.doRx(rcvd)
			if err != nil {
				log.Warn(err)
			}
		}
	}
}

func (sock *Sock) doRx(rcvd *Packet) error {
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

		err := fmt.Errorf("connection in unknown state")
		log.WithFields(log.Fields{
			"flag": flag,
			"pkt":  rcvd,
		}).Panic(err)
		return err
	}

	rcvd.Payload = rcvd.Payload[:rcvd.Length]

	sock.mux.Lock()
	sock.handleAck(rcvd)
	sock.handleData(rcvd)
	sock.mux.Unlock()

	log.WithFields(log.Fields{
		"sock.name":  sock.name,
		"pkt.seqNo":  rcvd.SeqNo,
		"pkt.ackNo":  rcvd.AckNo,
		"pkt.length": rcvd.Length,
	}).Info("received packet")

	return nil
}

// process ack
// sender
func (sock *Sock) handleAck(rcvd *Packet) {
	firstUnacked, err := sock.inFlight.start()
	if err != nil {
		firstUnacked = 0
	}

	if rcvd.AckNo >= firstUnacked {
		lastAcked, rtt := sock.inFlight.rcvdPkt(time.Now(), rcvd)
		if lastAcked == sock.lastAckedSeqNo && sock.nextSeqNo > lastAcked {
			sock.dupAckCnt++
			log.WithFields(log.Fields{
				"name":       sock.name,
				"lastAcked":  lastAcked,
				"rcvd.ackno": rcvd.AckNo,
				"inFlight":   sock.inFlight.order,
				"dupAcks":    sock.dupAckCnt,
			}).Info("dup ack")

			if sock.dupAckCnt >= 3 {
				// dupAckCnt >= 3 -> packet drop
				log.WithFields(log.Fields{
					"name":           sock.name,
					"sock.dupAckCnt": sock.dupAckCnt,
					"sock.lastAcked": sock.lastAckedSeqNo,
				}).Info("drop detected")
				sock.inFlight.drop(sock.lastAckedSeqNo)
				sock.dupAckCnt = 0
			}

			select {
			case sock.shouldTx <- struct{}{}:
			default:
			}
			return
		} else {
			sock.dupAckCnt = 0
		}

		sock.lastAckedSeqNo = lastAcked
		select {
		case sock.shouldTx <- struct{}{}:
		default:
		}

		log.WithFields(log.Fields{
			"name":       sock.name,
			"lastAcked":  sock.lastAckedSeqNo,
			"rcvd.ackno": rcvd.AckNo,
			"inFlight":   sock.inFlight.order,
			"rtt":        rtt,
		}).Info("new ack")

		sock.notifyAcks()
	}
}

// process received payload
func (sock *Sock) handleData(rcvd *Packet) {
	if rcvd.Length > 0 && rcvd.SeqNo >= sock.lastAck { // relevant data packet
		if _, ok := sock.rcvWindow.pkts[rcvd.SeqNo]; ok {
			// spurious retransmission
			return
		}

		// new data!
		sock.rcvWindow.addPkt(time.Now(), rcvd)
		ackNo, err := sock.rcvWindow.cumAck(sock.lastAck)
		if err != nil {
			ackNo = sock.lastAck
			log.WithFields(log.Fields{"name": sock.name, "ackNo": ackNo}).Warn(err)
		}

		sock.lastAck = ackNo
		log.WithFields(log.Fields{
			"name":         sock.name,
			"rcvd.seqno":   rcvd.SeqNo,
			"rcvd.length":  rcvd.Length,
			"sock.lastAck": sock.lastAck,
		}).Info("new data")

		copy(sock.readBuf[rcvd.SeqNo:], rcvd.Payload[:rcvd.Length])
		select {
		case sock.shouldPass <- sock.lastAck:
			log.Debug("sent on shouldPass")
		case <-sock.closed:
			close(sock.shouldPass)
			return
		default:
			log.Debug("skipping shouldPass")
		}

		select {
		case sock.shouldTx <- struct{}{}:
		default:
		}
	}
}
