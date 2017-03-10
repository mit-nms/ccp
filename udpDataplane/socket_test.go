package udpDataplane

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
)

func TestBasicTransfer(t *testing.T) {
	fmt.Println("starting...")
	// create dummy ccp
	ccp, err := ipc.SetupCcpListen()
	if err != nil {
		t.Error(err)
		return
	}

	go dummyCcp(ccp)
	testTransfer(t, "40000", bytes.Repeat([]byte{'a'}, 100), true)
	ccp.Close()
}

func TestLongerTransfer(t *testing.T) {
	// create dummy ccp
	ccp, err := ipc.SetupCcpListen()
	if err != nil {
		t.Error(err)
		return
	}

	go dummyCcp(ccp)
	testTransfer(t, "40001", bytes.Repeat([]byte{'b'}, 10000), true)
	ccp.Close()
}

func TestLongerTransferBigCwnd(t *testing.T) {
	// create dummy ccp
	ccp, err := ipc.SetupCcpListen()
	if err != nil {
		t.Error(err)
		return
	}

	go dummyCcp(ccp)
	testTransfer(t, "40002", bytes.Repeat([]byte{'c'}, 20000), false)
	ccp.Close()
}

func dummyCcp(ccp *ipc.Ipc) {
	go func() {
		ch, err := ccp.ListenCreateMsg()
		if err != nil {
			log.Error(err)
			return
		}
		for m := range ch {
			log.WithFields(log.Fields{
				"msg": m,
			}).Info("got msg")
		}
	}()

	go func() {
		ackCh, err := ccp.ListenAckMsg()
		if err != nil {
			log.Error(err)
			return
		}
		for ack := range ackCh {
			log.WithFields(log.Fields{
				"msg": ack,
			}).Info("got msg")
		}
	}()
}

func testTransfer(t *testing.T, port string, data []byte, smallcwnd bool) {
	done := make(chan error)
	cleanup := make(chan interface{})
	fmt.Println("starting receiver...")
	go receiver(port, data, done, cleanup, smallcwnd)
	<-time.After(time.Duration(100) * time.Millisecond)
	fmt.Println("starting sender...")
	go sender(port, data, done, cleanup, smallcwnd)

	err := <-done
	if err != nil {
		t.Error(err)
	}
	cleanup <- nil
}

func sender(port string, data []byte, done chan error, cleanup chan interface{}, smallcwnd bool) {
	sock, err := Socket("127.0.0.1", port, "SENDER")
	if err != nil {
		done <- err
		return
	}

	if smallcwnd {
		sock.cwnd = 1 * PACKET_SIZE
	} else {
		sock.cwnd = 5 * PACKET_SIZE
	}

	sent, err := sock.Write(data)
	if err != nil {
		done <- err
	}

	for a := range sent {
		log.WithFields(log.Fields{"acked": a, "tot": len(data)}).Debug("data acked")
		if a >= uint32(len(data)) {
			log.WithFields(log.Fields{"acked": a, "tot": len(data)}).Debug("stopping")
			done <- nil
			break
		}
	}

	<-cleanup
	sock.Close()
}

func receiver(port string, expect []byte, done chan error, cleanup chan interface{}, smallcwnd bool) {
	sock, err := Socket("", port, "RCVR")
	if err != nil {
		done <- err
		return
	}

	sock.mux.Lock()
	if smallcwnd {
		sock.cwnd = 1 * PACKET_SIZE
	} else {
		sock.cwnd = 5 * PACKET_SIZE
	}
	sock.mux.Unlock()

	rcvd := sock.Read(10)
	var b bytes.Buffer
	start := time.Now()
	for r := range rcvd {
		b.Write(r)
		fmt.Printf("%v: got %d/%d\n", time.Since(start), b.Len(), len(expect))
		if b.Len() >= len(expect) || time.Since(start) > time.Duration(5)*time.Second {
			break
		}
	}

	buf := b.Bytes()
	if len(buf) != len(expect) {
		done <- fmt.Errorf("received data doesn't match length: \n%v\n%v", buf, expect)
	}

	for i := 0; i < len(buf); i++ {
		if buf[i] != expect[i] {
			done <- fmt.Errorf("received data doesn't match: \n%v\n%v", buf, expect)
			break
		}
	}

	<-cleanup
	sock.Close()
}
