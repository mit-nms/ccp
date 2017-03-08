package main

import (
	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
)

var flows map[uint32]Flow

func main() {
	com, err := ipc.SetupCcpListen()

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
