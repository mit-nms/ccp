package main

import (
	"ccp/ccpFlow"
	"ccp/ipc"

	log "github.com/sirupsen/logrus"
)

func handleMsgs(
	measureCh chan ipc.MeasureMsg,
	createCh chan ipc.CreateMsg,
	dropCh chan ipc.DropMsg,
) {
	for {
		select {
		case m := <-measureCh:
			handleMeasure(m)
		case cr := <-createCh:
			handleCreate(cr)
		case dr := <-dropCh:
			handleDrop(dr)
		}
	}
}

func handleMeasure(ack ipc.MeasureMsg) {
	log.WithFields(log.Fields{
		"flowid": ack.SocketId(),
		"ackno":  ack.AckNo(),
		"rtt":    ack.Rtt(),
		"rin":    ack.Rin(),
		"rout":   ack.Rout(),
	}).Debug("handleMeasure")

	if flow, ok := flows[ack.SocketId()]; !ok {
		log.WithFields(log.Fields{"flowid": ack.SocketId()}).Warn("Unknown flow")
		return
	} else {
		flow.GotMeasurement(ccpFlow.Measurement{
			Ack:  ack.AckNo(),
			Rtt:  ack.Rtt(),
			Rin:  ack.Rin(),
			Rout: ack.Rout(),
		})
	}
}

func handleDrop(dr ipc.DropMsg) {
	log.WithFields(log.Fields{
		"flowid":  dr.SocketId,
		"drEvent": dr.Event,
	}).Debug("handleDrop")

	if flow, ok := flows[dr.SocketId()]; !ok {
		log.WithFields(log.Fields{"flowid": dr.SocketId()}).Warn("Unknown flow")
		return
	} else {
		flow.Drop(ccpFlow.DropEvent(dr.Event()))
	}
}

func handleCreate(cr ipc.CreateMsg) {
	log.WithFields(log.Fields{
		"flowid":   cr.SocketId(),
		"startseq": cr.StartSeq(),
		"alg":      cr.CongAlg(),
	}).Info("handleCreate")
	if _, ok := flows[cr.SocketId()]; ok {
		log.WithFields(log.Fields{"flowid": cr.SocketId()}).Error("Creating already created flow")
		return
	}

	var f ccpFlow.Flow
	var err error
	if *overrideAlg != "nil" {
		f, err = ccpFlow.GetFlow(*overrideAlg)
		if err != nil {
			log.WithFields(log.Fields{
				"datapath request": cr.CongAlg(),
				"override request": *overrideAlg,
				"error":            err,
			}).Warn("Unknown flow type, trying datapath request")
		} else {
			goto gotFlow
		}
	}

	f, err = ccpFlow.GetFlow(cr.CongAlg())
	if err != nil {
		log.WithFields(log.Fields{
			"alg":   cr.CongAlg(),
			"error": err,
		}).Warn("Unknown flow type, using reno")
		f, err = ccpFlow.GetFlow("reno")
		if err != nil {
			log.WithFields(log.Fields{
				"registered": ccpFlow.ListRegistered(),
				"asked":      "reno",
			}).Panic(err)
		}
	}

gotFlow:
	ipCh, err := ipc.SetupCcpSend(dp, cr.SocketId())
	if err != nil {
		log.WithFields(log.Fields{"flowid": cr.SocketId()}).Error("Error creating ccp->socket ipc channel for flow")
	}

	switch dp {
	case ipc.UNIX:
		f.Create(cr.SocketId(), ipCh, 1462, cr.StartSeq(), uint32(*initCwnd))
	case ipc.NETLINK:
		f.Create(cr.SocketId(), ipCh, 1460, cr.StartSeq(), uint32(*initCwnd))
	}

	flows[cr.SocketId()] = f
}
