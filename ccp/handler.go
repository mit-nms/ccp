package main

import (
	"ccp/ccpFlow"
	"ccp/ipc"
	"ccp/ipcBackend"

	log "github.com/Sirupsen/logrus"
)

func handleMsgs(
	ackCh chan ipcbackend.AckMsg,
	createCh chan ipcbackend.CreateMsg,
	dropCh chan ipcbackend.DropMsg,
) {
	for {
		select {
		case ack := <-ackCh:
			handleAck(ack)
		case cr := <-createCh:
			handleCreate(cr)
		case dr := <-dropCh:
			handleDrop(dr)
		}
	}
}

func handleAck(ack ipcbackend.AckMsg) {
	log.WithFields(log.Fields{
		"flowid": ack.SocketId(),
		"ackno":  ack.AckNo(),
		"rtt":    ack.Rtt(),
	}).Info("handleAck")

	if flow, ok := flows[ack.SocketId()]; !ok {
		log.WithFields(log.Fields{"flowid": ack.SocketId()}).Warn("Unknown flow")
		return
	} else {
		flow.Ack(ack.AckNo(), ack.Rtt())
	}
}

func handleDrop(dr ipcbackend.DropMsg) {
	log.WithFields(log.Fields{
		"flowid":  dr.SocketId,
		"drEvent": dr.Event,
	}).Info("handleDrop")

	if flow, ok := flows[dr.SocketId()]; !ok {
		log.WithFields(log.Fields{"flowid": dr.SocketId()}).Warn("Unknown flow")
		return
	} else {
		flow.Drop(ccpFlow.DropEvent(dr.Event()))
	}
}

func handleCreate(cr ipcbackend.CreateMsg) {
	log.WithFields(log.Fields{
		"flowid": cr.SocketId(),
		"alg":    cr.CongAlg(),
	}).Info("handleCreate")
	if _, ok := flows[cr.SocketId()]; ok {
		log.WithFields(log.Fields{"flowid": cr.SocketId()}).Error("Creating already created flow")
		return
	}

	f, err := ccpFlow.GetFlow(cr.CongAlg())
	if err != nil {
		log.WithFields(log.Fields{"alg": cr.CongAlg()}).Warn("Unknown flow type, using reno")
		f, err = ccpFlow.GetFlow("reno")
		if err != nil {
			log.WithFields(log.Fields{
				"registered": ccpFlow.ListRegistered(),
				"asked":      "reno",
			}).Panic(err)
		}
	}

	ipCh, err := ipc.SetupCcpSend(dp, cr.SocketId())
	if err != nil {
		log.WithFields(log.Fields{"flowid": cr.SocketId()}).Error("Error creating ccp->socket ipc channel for flow")
	}

	f.Create(cr.SocketId(), ipCh)
	flows[cr.SocketId()] = f
}
