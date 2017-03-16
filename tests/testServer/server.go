package main

import (
	"bytes"
	"strconv"
	"strings"

	"ccp/udpDataplane"

	log "github.com/Sirupsen/logrus"
)

func main() {
	sock, err := udpDataplane.Socket("", "40000", "SERVER")
	if err != nil {
		log.Error(err)
		return
	}

	rcvd := sock.Read(1)
	rb := <-rcvd
	req := strings.Split(string(rb), " ")
	if len(req) != 3 || req[0] != "testRequest:" || req[1] != "size" {
		log.WithFields(log.Fields{
			"got":      rb,
			"expected": "'test request: size {x}'",
		}).Warn("malformed request")
	}

	sz, err := strconv.Atoi(req[2])
	if err != nil {
		log.Error(err)
		return
	}

	log.WithFields(log.Fields{
		"req": string(rb),
	}).Info("got req")

	resp := bytes.Repeat([]byte("test response\n"), sz/14)
	sent, err := sock.Write(resp)
	if err != nil {
		log.Error(err)
		return
	}

	log.WithFields(log.Fields{
		"len":       len(resp),
		"asked len": sz,
	}).Info("started write")

	for a := range sent {
		log.WithFields(log.Fields{
			"ack":   a,
			"total": len(resp),
		}).Info("server acked")
	}
}
