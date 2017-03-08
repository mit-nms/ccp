package main

import (
	"ccp/ipc"
)

type Flow interface {
	Create(send ipc.SendOnly)
	Ack(ack uint32)
}

func getFlow(alg string) Flow {
	return nil
}
