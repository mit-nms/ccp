package main

import (
	"ccp/ccpFlow"
	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
)

var flows map[uint32]ccpFlow.Flow

func main() {
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

	handleMsgs(ackCh, createCh)
}
