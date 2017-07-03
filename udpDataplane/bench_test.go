package udpDataplane

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"ccp/ccpFlow"
	"ccp/ipc"
	"ccp/reno"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.WarnLevel)
}

func BenchmarkDoRx(b *testing.B) {
	// setup dummy socket
	s := &Sock{
		name: "test",

		conn:     nil,
		writeBuf: make([]byte, 2e6),
		readBuf:  make([]byte, 2e6),

		//sender
		cwnd:           5 * PACKET_SIZE, // init_cwnd = ~ 1 pkt
		lastAckedSeqNo: 0,
		dupAckCnt:      0,
		nextSeqNo:      0,
		inFlight:       makeWindow(),

		//receiver
		lastAck:   0,
		rcvWindow: makeWindow(),

		// ccp communication
		ackNotifyThresh: PACKET_SIZE * 10, // ~ 10 pkts

		// synchronization
		shouldTx:    make(chan interface{}, 1),
		shouldPass:  make(chan uint32, 1),
		notifyAcks:  make(chan notifyAck, 1),
		notifyDrops: make(chan notifyDrop, 1),
		ackedData:   make(chan uint32),
		closed:      make(chan interface{}),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		doBenchmarkRx(s)
		s.inFlight = makeWindow()
	}
}

func doBenchmarkRx(s *Sock) {
	// check sender side (receiving acks)
	s.inFlight.addPkt(time.Now(), &Packet{
		SeqNo:   uint32(1460),
		AckNo:   0,
		Flag:    ACK,
		Length:  1462,
		Payload: bytes.Repeat([]byte("t"), 1462),
	})

	s.doRx(&Packet{
		SeqNo:  0,
		AckNo:  uint32(1460),
		Flag:   ACK,
		Length: 0,
	})
}

func BenchmarkSocket(b *testing.B) {
	b.SetBytes(1e6)
	for i := 0; i < b.N; i++ {
		kill := make(chan interface{})
		ready := make(chan interface{})
		go renoCcp(kill, ready)
		<-ready
		log.Debug("ccp ready")
		go server(kill, ready)
		<-ready
		done := make(chan interface{})
		go client(done)
		<-done
		close(kill)
	}
}

// from ccp/ccp.go
func renoCcp(killed chan interface{}, ready chan interface{}) {
	com, err := ipc.SetupCcpListen(ipc.UDP)
	if err != nil {
		log.Error(err)
		return
	}

	ackCh, err := com.ListenMeasureMsg()
	if err != nil {
		log.Error(err)
		return
	}

	dropCh, err := com.ListenDropMsg()
	if err != nil {
		log.Error(err)
		return
	}

	createCh, err := com.ListenCreateMsg()
	if err != nil {
		log.Error(err)
		return
	}

	ready <- struct{}{}

	r := &reno.Reno{}
	for {
		select {
		case <-killed:
			return
		case cr := <-createCh:
			log.Info("got create")
			ipCh, err := ipc.SetupCcpSend(ipc.UDP, cr.SocketId())
			if err != nil {
				log.WithFields(log.Fields{"flowid": cr.SocketId()}).Error("Error creating ccp->socket ipc channel for flow")
			}
			r.Create(40000, ipCh, 1462, 0, 10)
		case ack := <-ackCh:
			log.Info("got ack")
			r.Ack(ack.AckNo(), ack.Rtt())
		case dr := <-dropCh:
			log.Info("got drop")
			r.Drop(ccpFlow.DropEvent(dr.Event()))
		}
	}
}

// from test/testServer/server.go
func server(killed chan interface{}, ready chan interface{}) {
	sockCh := socketNonBlocking("", "40000", "SERVER")
	ready <- struct{}{}
	sock := <-sockCh

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
		select {
		case <-killed:
			return
		default:
		}

		log.WithFields(log.Fields{
			"ack":   a,
			"total": len(resp),
		}).Info("server acked")
	}
}

// from test/testClient/client.go
func client(done chan interface{}) {
	size := int(1e6)
	sock, err := Socket("127.0.0.1", "40000", "CLIENT")
	if err != nil {
		log.Error(err)
		return
	}

	req := []byte(fmt.Sprintf("testRequest: size %d", size))
	_, err = sock.Write(req)
	if err != nil {
		log.Error(err)
		return
	}

	response := new(bytes.Buffer)
	rcvd := sock.Read(100)
	for rb := range rcvd {
		response.Write(rb)
		if response.Len() >= size-13 {
			break
		}
	}

	log.WithFields(log.Fields{
		"got": response.Len(),
	}).Info("done")

	sock.Fin()

	done <- struct{}{}
}
