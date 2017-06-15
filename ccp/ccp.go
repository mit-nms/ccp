package main

import (
	"flag"

	"ccp/ccpFlow"
	"ccp/cubic"
	"ccp/ipc"
	"ccp/reno"
	"ccp/vegas"

	log "github.com/Sirupsen/logrus"
)

var datapath = flag.String("datapath", "udp", "which IPC backend to use (udp|kernel)")

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
		"datapath": *datapath,
	}).Warn("datapath")
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

	ackCh, err := com.ListenAckMsg()
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
