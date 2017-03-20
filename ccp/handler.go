package main

import (
	"ccp/ccpFlow"
	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
)

func handleMsgs(ackCh chan ipc.AckMsg, createCh chan ipc.CreateMsg) {
	for {
		select {
		case ack := <-ackCh:
			handleAck(ack)
		case cr := <-createCh:
			handleCreate(cr)
		}
	}
}

func handleAck(ack ipc.AckMsg) {
	log.WithFields(log.Fields{
		"flowid": ack.SocketId,
		"ackno":  ack.AckNo,
	}).Info("handleAck")
	if flow, ok := flows[ack.SocketId]; !ok {
		log.WithFields(log.Fields{"flowid": ack.SocketId}).Warn("Unknown flow")
		return
	} else {
		flow.Ack(ack.AckNo)
	}
}

func handleCreate(cr ipc.CreateMsg) {
	log.WithFields(log.Fields{
		"flowid": cr.SocketId,
		"alg":    cr.CongAlg,
	}).Info("handleCreate")
	if _, ok := flows[cr.SocketId]; ok {
		log.WithFields(log.Fields{"flowid": cr.SocketId}).Error("Creating already created flow")
		return
	}

	f, err := ccpFlow.GetFlow(cr.CongAlg)
	if err != nil {
		log.WithFields(log.Fields{"alg": cr.CongAlg}).Warn("Unknown flow type, using reno")
		f, _ = ccpFlow.GetFlow("reno")
	}

	flows[cr.SocketId] = f

	ipCh, err := ipc.SetupCcpSend(cr.SocketId)
	if err != nil {
		log.WithFields(log.Fields{"flowid": cr.SocketId}).Error("Error creating ccp->socket ipc channel for flow")
	}

	flows[cr.SocketId].Create(cr.SocketId, ipCh)
}
