// +build linux

package main

import (
	"bytes"
	"fmt"
	"time"

	"ccp/ipcBackend"
	"ccp/netlinkipc"

	log "github.com/sirupsen/logrus"
)

func main() {
	nl, err := netlinkipc.New().SetupSend("", 33).SetupListen("", 33).SetupFinish()
	if err != nil {
		log.WithFields(log.Fields{
			"where": "recv setup",
		}).Warn(err)
		return
	}

	msgs := nl.Listen()
	testControl(nl, msgs)
}

func testControl(nl ipcbackend.Backend, msgs chan ipcbackend.Msg) {
	cwnd := uint32(10) * 1460
	for msg := range msgs {
		switch msg.(type) {
		case ipcbackend.CreateMsg:
			cr := msg.(ipcbackend.CreateMsg)
			log.WithFields(log.Fields{
				"got": fmt.Sprintf("(%v, %v)", cr.SocketId(), cr.CongAlg()),
			}).Info("rcvd create")
		case ipcbackend.AckMsg:
			ack := msg.(ipcbackend.AckMsg)
			log.WithFields(log.Fields{
				"got":  fmt.Sprintf("(%v, %v, %v)", ack.SocketId(), ack.AckNo(), ack.Rtt()),
				"cwnd": cwnd,
			}).Info("rcvd ack")

			cwnd += 2 * 1460

			cwMsg := nl.GetCwndMsg()
			cwMsg.New(ack.SocketId(), cwnd)

			<-time.After(time.Second)

			nl.SendMsg(cwMsg)
			log.WithFields(log.Fields{
				"sent": fmt.Sprintf("(%v, %v)", cwMsg.SocketId(), cwMsg.Cwnd()),
				"cwnd": cwnd,
			}).Info("sent cwnd msg")
		default:
			log.WithFields(log.Fields{
				"msg": msg,
			}).Warn("bad message type")
		}
	}
}

func logMsgs(msgs chan ipcbackend.Msg) {
	for msg := range msgs {
		switch msg.(type) {
		case ipcbackend.CreateMsg:
			cr := msg.(ipcbackend.CreateMsg)
			log.WithFields(log.Fields{
				"got": fmt.Sprintf("(%v, %v)", cr.SocketId(), cr.CongAlg()),
			}).Info("rcvd create")
		case ipcbackend.AckMsg:
			ack := msg.(ipcbackend.AckMsg)
			log.WithFields(log.Fields{
				"got": fmt.Sprintf("(%v, %v, %v)", ack.SocketId(), ack.AckNo(), ack.Rtt()),
			}).Info("rcvd ack")
		default:
			log.WithFields(log.Fields{
				"msg": msg,
			}).Warn("bad message type")
		}
	}
}

func test(nl ipcbackend.Backend, msgs chan ipcbackend.Msg) {
	msg := <-msgs

	log.WithFields(log.Fields{
		"msg": msg,
	}).Info("got msg")
	err := expectCreate(msg, 42, "reno")
	if err != nil {

	}

	cwMsg := nl.GetCwndMsg()
	cwMsg.New(42, 4380)
	nl.SendMsg(cwMsg)

	msg = <-msgs

	log.WithFields(log.Fields{
		"msg": msg,
	}).Info("got msg")
	err = expectAck(msg, 42, 1461, time.Duration(255)*time.Nanosecond)
}

func expectCreate(msg ipcbackend.Msg, sid uint32, alg string) error {
	switch msg.(type) {
	case ipcbackend.CreateMsg:
		cr := msg.(ipcbackend.CreateMsg)
		if cr.SocketId() == sid && bytes.Equal([]byte(cr.CongAlg()), []byte(alg)) {
			log.WithFields(log.Fields{
				"got": fmt.Sprintf("(%v, %v)", cr.SocketId(), cr.CongAlg()),
			}).Info("ok")
		} else {
			log.WithFields(log.Fields{
				"expected":     fmt.Sprintf("(%v, %v)", sid, alg),
				"got":          fmt.Sprintf("(%v, %v)", cr.SocketId(), cr.CongAlg()),
				"alg bytes":    []byte(cr.CongAlg()),
				"expect bytes": []byte(alg),
			}).Warn("bad message")
			return fmt.Errorf("bad message")
		}
	default:
		log.WithFields(log.Fields{
			"expected": fmt.Sprintf("(%v, %v)", sid, alg),
		}).Warn("bad message type")
		return fmt.Errorf("bad message type")
	}

	return nil
}

func expectAck(msg ipcbackend.Msg, sid uint32, ackNo uint32, rtt time.Duration) error {
	switch msg.(type) {
	case ipcbackend.AckMsg:
		ack := msg.(ipcbackend.AckMsg)
		if ack.SocketId() == sid && ack.AckNo() == ackNo {
			log.WithFields(log.Fields{
				"got": fmt.Sprintf("(%v, %v, %v)", ack.SocketId(), ack.AckNo(), ack.Rtt()),
			}).Info("ok")
		} else {
			log.WithFields(log.Fields{
				"expected": fmt.Sprintf("(%v, %v, %v)", sid, ackNo, rtt),
				"got":      fmt.Sprintf("(%v, %v, %v)", ack.SocketId(), ack.AckNo(), ack.Rtt()),
			}).Warn("bad message")
			return fmt.Errorf("bad message")
		}
	default:
		log.WithFields(log.Fields{
			"expected": fmt.Sprintf("(%v, %v, %v)", sid, ackNo, rtt),
		}).Warn("bad message type")
		return fmt.Errorf("bad message type")
	}

	return nil
}
