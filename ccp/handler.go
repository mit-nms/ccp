package main

import (
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
	if flow, ok := flows[ack.SocketId]; !ok {
		log.WithFields(log.Fields{"flowid": ack.SocketId}).Warn("Unknown flow")
		return
	} else {
		flow.Ack(ack.AckNo)
	}
}

func handleCreate(cr ipc.CreateMsg) {
	if _, ok := flows[cr.SocketId]; ok {
		log.WithFields(log.Fields{"flowid": cr.SocketId}).Error("Creating already created flow")
		return
	}

	flows[cr.SocketId] = getFlow(cr.CongAlg)

	ipCh, err := ipc.SetupCcpSend(cr.SocketId)
	if err != nil {
		log.WithFields(log.Fields{"flowid": cr.SocketId}).Error("Error creating ccp->socket ipc channel for flow")
	}

	flows[cr.SocketId].Create(ipCh)
}
