// +build linux

package main

import (
	"bytes"
	"fmt"
	"time"

	"ccp/ipcBackend"
	"ccp/netlinkipc"

	log "github.com/Sirupsen/logrus"
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
	msg := <-msgs

	log.WithFields(log.Fields{
		"msg": msg,
	}).Info("got msg")
	err = expectCreate(msg, 42, "reno")
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
