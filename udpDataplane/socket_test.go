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
	cleanup := make(chan *Sock)
	exit := make(chan interface{})
	fmt.Println("starting receiver...")
	go receiver(port, data, done, cleanup, exit)
	<-time.After(time.Duration(100) * time.Millisecond)
	fmt.Println("starting sender...")
	go sender(port, data, done, cleanup, exit, smallcwnd)

	err := <-done
	if err != nil {
		t.Error(err)
	}

	s1 := <-cleanup
	s1.Close()
	fmt.Printf("closed %s\n", s1.name)
	for {
		select {
		case s2 := <-cleanup:
			s2.Close()
			return
		default:
			select {
			case exit <- struct{}{}:
			default:
			}
		}
	}
}

func sender(port string, data []byte, done chan error, cleanup chan *Sock, exit chan interface{}, smallcwnd bool) {
	sock, err := Socket("127.0.0.1", port, "SENDER")
	if err != nil {
		select {
		case done <- err:
		default:
		}
		return
	}

	if smallcwnd {
		sock.cwnd = 1 * PACKET_SIZE
	} else {
		sock.cwnd = 5 * PACKET_SIZE
	}

	sent, err := sock.Write(data)
	if err != nil {
		select {
		case done <- err:
		default:
		}
	}

loop:
	for {
		select {
		case a := <-sent:
			log.WithFields(log.Fields{"acked": a, "tot": len(data)}).Debug("data acked")
			if a >= uint32(len(data)) {
				log.WithFields(log.Fields{"acked": a, "tot": len(data)}).Debug("stopping")
				select {
				case done <- nil:
				default:
				}
				break loop
			}
		case <-exit:
			break loop
		case <-time.After(3 * time.Second):
			select {
			case done <- fmt.Errorf("test timeout"):
			default:
				break loop
			}
		}
	}

	cleanup <- sock
}

func receiver(port string, expect []byte, done chan error, cleanup chan *Sock, exit chan interface{}) {
	sock, err := Socket("", port, "RCVR")
	if err != nil {
		done <- err
		return
	}

	var b bytes.Buffer
	buf := b.Bytes()
	rcvd := sock.Read(10)
	start := time.Now()
loop:
	for {
		select {
		case r := <-rcvd:
			b.Write(r)
			log.WithFields(log.Fields{
				"time": time.Since(start),
				"got":  b.Len(),
				"tot":  len(expect),
			}).Info("app receiver")
			if b.Len() >= len(expect) || time.Since(start) > time.Duration(5)*time.Second {
				log.WithFields(log.Fields{
					"time": time.Since(start),
					"got":  b.Len(),
					"tot":  len(expect),
				}).Info("app receiver stopping")
				break loop
			}
		case <-exit:
			break loop
		case <-time.After(3 * time.Second):
			break loop
		}
	}

	err = nil
	if b.Len() != len(expect) {
		err = fmt.Errorf("received data doesn't match length: \n%v\n%v", len(buf), len(expect))
		goto cl
	}

	if !bytes.Equal(b.Bytes(), expect) {
		err = fmt.Errorf("received data doesn't match: \n%v\n%v", len(buf), len(expect))
	}

cl:
	select {
	case done <- err:
	default:
	}

	cleanup <- sock
}
