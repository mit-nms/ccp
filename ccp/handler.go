package main

import (
	"ccp/ccpFlow"
	"ccp/ipc"

	log "github.com/sirupsen/logrus"
)

type flowHandler struct {
	flowMeasureCh chan ipc.MeasureMsg
	flowDropCh    chan ipc.DropMsg
}

/* The event loop for the CCP
 * Demultiplex messages across flows, and dispatch new per-flow
 * event loops on CREATE messages.
 */
func handleMsgs(
	createCh chan ipc.CreateMsg,
	measureCh chan ipc.MeasureMsg,
	dropCh chan ipc.DropMsg,
) {
	endFlow := make(chan uint32)
	for {
		select {
		case cr := <-createCh:
			handleCreate(cr, endFlow)
		case m := <-measureCh:
			handleMeasure(m)
		case dr := <-dropCh:
			handleDrop(dr)
		case sid := <-endFlow:
			handleFlowEnd(sid)
		}
	}
}

func handleCreate(cr ipc.CreateMsg, endFlow chan uint32) {
	log.WithFields(log.Fields{
		"flowid":   cr.SocketId(),
		"startseq": cr.StartSeq(),
		"alg":      cr.CongAlg(),
	}).Info("handleCreate")

	if _, ok := flows[cr.SocketId()]; ok {
		log.WithFields(log.Fields{
			"flowid": cr.SocketId(),
		}).Error("Creating already created flow")
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
		log.WithFields(log.Fields{
			"flowid": cr.SocketId(),
		}).Error("Error creating ccp->socket ipc channel for flow")
	}

	switch dp {
	case ipc.UNIX:
		f.Create(cr.SocketId(), ipCh, 1462, cr.StartSeq(), uint32(*initCwnd))
	case ipc.NETLINK:
		f.Create(cr.SocketId(), ipCh, 1460, cr.StartSeq(), uint32(*initCwnd))
	}

	handler := flowHandler{
		flowMeasureCh: make(chan ipc.MeasureMsg),
		flowDropCh:    make(chan ipc.DropMsg),
	}

	go handleFlow(cr.SocketId(), f, handler, endFlow)
	flows[cr.SocketId()] = handler
}

func handleMeasure(m ipc.MeasureMsg) {
	if handler, ok := flows[m.SocketId()]; !ok {
		log.WithFields(log.Fields{
			"flowid": m.SocketId(),
			"msg":    "measure",
		}).Warn("Unknown flow")
		return
	} else {
		handler.flowMeasureCh <- m
	}
}

func handleDrop(dr ipc.DropMsg) {
	if handler, ok := flows[dr.SocketId()]; !ok {
		log.WithFields(log.Fields{
			"flowid": dr.SocketId(),
			"msg":    "drop",
		}).Warn("Unknown flow")
		return
	} else {
		handler.flowDropCh <- dr
	}
}

func handleFlowEnd(sid uint32) {
	delete(flows, sid)
}
