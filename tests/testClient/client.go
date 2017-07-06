package main

import (
	"bytes"
	"flag"
	"fmt"

	"ccp/udpDataplane"

	log "github.com/sirupsen/logrus"
)

var ip = flag.String("ip", "127.0.0.1", "ip address to connect to")
var size = flag.Int("size", 11200, "amount of data to request")

func main() {
	flag.Parse()
	sock, err := udpDataplane.Socket(*ip, "40000", "CLIENT")
	if err != nil {
		log.Error(err)
		return
	}

	req := []byte(fmt.Sprintf("testRequest: size %d", *size))
	_, err = sock.Write(req)
	if err != nil {
		log.Error(err)
		return
	}

	response := new(bytes.Buffer)
	rcvd := sock.Read(100)
	for rb := range rcvd {
		response.Write(rb)
		if response.Len() >= *size-13 {
			break
		}
	}

	log.WithFields(log.Fields{
		"got": response.Len(),
	}).Info("done")

	sock.Fin()
}
