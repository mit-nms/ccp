package ccpFlow

import (
	"fmt"
	"time"

	"ccp/ipc"
)

type DropEvent string

var DupAck DropEvent = DropEvent("dupack")
var Timeout DropEvent = DropEvent("timeout")
var Ecn DropEvent = DropEvent("ecn")

type Measurement struct {
	Ack  uint32
	Rtt  time.Duration
	Rin  uint64
	Rout uint64
	Loss uint32
}

type Flow interface {
	// Name returns a string identifying the CC algorithm
	Name() string
	// Create takes configuration parameters from the CCP and does initialization
	Create(
		sockid uint32,
		send ipc.SendOnly,
		pktsz uint32,
		startSeq uint32,
		initCwnd uint32,
	)
	// Measurement: callback for when a specified measurement is received
	GotMeasurement(m Measurement)
	// Drop: callback for drop event
	Drop(event DropEvent)
}

// name of flow to function which returns blank instance
var protocolRegistry map[string]func() Flow

// Register a new type of flow
// name: unique name of the flow type
// f: function which returns a blank instance of an implementing type
func Register(name string, f func() Flow) error {
	if protocolRegistry == nil {
		protocolRegistry = make(map[string]func() Flow)
	}

	if _, ok := protocolRegistry[name]; ok {
		return fmt.Errorf("flow algorithm %v already registered", name)
	}
	protocolRegistry[name] = f
	return nil
}

func ListRegistered() (regs []string) {
	for name, _ := range protocolRegistry {
		regs = append(regs, name)
	}
	return
}

func GetFlow(name string) (Flow, error) {
	if f, ok := protocolRegistry[name]; !ok {
		return nil, fmt.Errorf("unknown flow algorithm %v", name)
	} else {
		return f(), nil
	}
}
