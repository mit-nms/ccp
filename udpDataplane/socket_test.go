package udpDataplane

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"ccp/ipcbackend"
	"ccp/unixsocket"

	log "github.com/Sirupsen/logrus"
)

var sndSock *Sock
var rcvSock *Sock

func TestBasicTransfer(t *testing.T) {
	fmt.Println("starting...")
	// create dummy ccp
	ccp, err := unixsocket.SetupCcp()
	if err != nil {
		t.Error(err)
		return
	}

	go dummyCcp(ccp)

	testTransfer(t, bytes.Repeat([]byte{'a'}, 100))

	os.Remove("/tmp/ccp-in")
}

func dummyCcp(ccp ipcbackend.Backend) {
	msgs, _ := ccp.ListenMsg()
	for msg := range msgs {
		log.WithFields(log.Fields{
			"msg": msg,
		}).Info("got msg")
	}
}

func testTransfer(t *testing.T, data []byte) {
	done := make(chan error)
	cleanup := make(chan interface{})
	fmt.Println("starting receiver...")
	go receiver(data, done, cleanup)
	<-time.After(time.Duration(100) * time.Millisecond)
	fmt.Println("starting sender...")
	go sender(data, done, cleanup)

	select {
	case err := <-done:
		t.Error(err)
		return
	case <-cleanup:
		sndSock.Close()
		rcvSock.Close()
		return
	}
}

func sender(data []byte, done chan error, cleanup chan interface{}) {
	sock, err := Socket("127.0.0.1", "60000", "SENDER")
	if err != nil {
		done <- err
		return
	}

	sndSock = sock

	sock.Write(data)
}

func receiver(expect []byte, done chan error, cleanup chan interface{}) {
	sock, err := Socket("", "60000", "RCVR")
	if err != nil {
		done <- err
		return
	}

	rcvSock = sock

	rcvd := sock.Read(10)
	var b bytes.Buffer
	for r := range rcvd {
		b.Write(r)
		if b.Len() >= 100 {
			break
		}
	}

	buf := b.Bytes()
	if len(buf) != len(expect) {
		done <- fmt.Errorf("received data doesn't match: \n%v\n%v", buf, expect)
		sock.Close()
		return
	}

	for i := 0; i < len(buf); i++ {
		if buf[i] != expect[i] {
			done <- fmt.Errorf("received data doesn't match: \n%v\n%v", buf, expect)
			sock.Close()
			return
		}
	}

	cleanup <- nil
}
