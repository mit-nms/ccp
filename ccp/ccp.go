package main

import (
	"ccp/ccpFlow"
	"ccp/ipc"
	"ccp/reno"

	log "github.com/Sirupsen/logrus"
)

func init() {
	log.SetLevel(log.InfoLevel)
}

var flows map[uint32]ccpFlow.Flow

func main() {
	flows = make(map[uint32]ccpFlow.Flow)
	reno.Init()

	com, err := ipc.SetupCcpListen()
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
