package udpDataplane

import (
	"bytes"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
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
		doBenchmark(s)
		s.inFlight = makeWindow()
	}
}

func doBenchmark(s *Sock) {
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
