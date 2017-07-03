package main

import (
	"flag"

	"ccp/ccpFlow"
	"ccp/cubic"
	"ccp/ipc"
	"ccp/reno"
	"ccp/vegas"

	log "github.com/sirupsen/logrus"
)

var datapath = flag.String("datapath", "udp", "which IPC backend to use (udp|kernel)")
var overrideAlg = flag.String("congAlg", "nil", "override the datapath's requested congestion control algorithm for all flows (cubic|reno|vegas|nil)")
var initCwnd = flag.Uint("initCwnd", 10, "override the default starting congestion window")

var flows map[uint32]ccpFlow.Flow
var dp ipc.Datapath

func init() {
	log.SetLevel(log.InfoLevel)
	flows = make(map[uint32]ccpFlow.Flow)
	cubic.Init()
	vegas.Init()
	reno.Init()
}

func main() {
	flag.Parse()

	log.WithFields(log.Fields{
		"datapath":    *datapath,
		"overrideAlg": *overrideAlg,
		"startCwnd":   *initCwnd,
	}).Info("parsed flags")

	switch *datapath {
	case "udp":
		dp = ipc.UDP
	case "kernel":
		dp = ipc.KERNEL
	default:
		log.WithFields(log.Fields{
			"datapath": *datapath,
		}).Warn("unknown datapath")
		return
	}

	com, err := ipc.SetupCcpListen(dp)
	if err != nil {
		log.Error(err)
		return
	}

	ackCh, err := com.ListenMeasureMsg()
	if err != nil {
		log.Error(err)
		return
	}

	createCh, err := com.ListenCreateMsg()
	if err != nil {
		log.Error(err)
		return
	}

	dropCh, err := com.ListenDropMsg()
	if err != nil {
		log.Error(err)
		return
	}

	handleMsgs(ackCh, createCh, dropCh)
}
