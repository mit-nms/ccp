package udpDataplane

import (
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
)

func TestAddWindow(t *testing.T) {
	w := makeWindow()

	for i := 0; i < 5; i++ {
		w.addPkt(time.Now(), &Packet{
			SeqNo:   uint32(i),
			AckNo:   0,
			Flag:    ACK,
			Length:  0,
			Payload: []byte{},
		})

		log.WithFields(log.Fields{
			"order": w.order,
			"pkts":  w.pkts,
			"new":   i,
		}).Info("addPkt")

		for i, seq := range w.order {
			if seq != uint32(i) {
				t.Errorf("expected seq %d, got seq %d in %v", i, seq, w)
				return
			}
		}
	}
}

func TestGotPacket(t *testing.T) {
	w := makeWindow()

	for i := 0; i < 6; i++ {
		w.addPkt(time.Now(), &Packet{
			SeqNo:   uint32(i),
			AckNo:   0,
			Flag:    ACK,
			Length:  1,
			Payload: []byte{'a'},
		})

		log.WithFields(log.Fields{
			"order": w.order,
			"pkts":  w.pkts,
			"new":   i,
		}).Info("addPkt")
	}

	s, err := w.start()
	if err != nil {
		t.Error(err)
		return
	}

	if s != 0 {
		t.Errorf("expected start 0, got %d", s)
		return
	}

	ack, err := w.cumAck(0)
	if err != nil {
		t.Error(err)
		return
	}

	if ack != 6 {
		t.Errorf("expected cum ack 6, got %d", ack)
		return
	}

	s, err = w.start()
	if err == nil {
		t.Errorf("expected error, got %v", s)
		return
	}
}

func TestRcvdPacket(t *testing.T) {
	w := makeWindow()

	for i := 0; i < 6; i++ {
		w.addPkt(time.Now(), &Packet{
			SeqNo:   uint32(i),
			AckNo:   0,
			Flag:    ACK,
			Length:  1,
			Payload: []byte{'a'},
		})

		log.WithFields(log.Fields{
			"order": w.order,
			"pkts":  w.pkts,
			"new":   i,
		}).Info("addPkt")
	}

	s, err := w.start()
	if err != nil {
		t.Error(err)
		return
	}

	if s != 0 {
		t.Errorf("expected start 0, got %d", s)
		return
	}

	cumAcked, _ := w.rcvdPkt(time.Now(), &Packet{
		SeqNo:   0,
		AckNo:   3,
		Flag:    ACK,
		Length:  0,
		Payload: []byte{'a'},
	})
	if err != nil {
		t.Error(err)
		return
	}

	if cumAcked != 3 {
		t.Errorf("expected cum ack 6, got %d", cumAcked)
		return
	}

	s, err = w.start()
	if err != nil {
		t.Error(err)
		return
	}

	if s != 3 {
		t.Errorf("expected start 3, got %d", s)
		return
	}
}

func TestOutOfOrderReceive(t *testing.T) {
	w := makeWindow()

	for i := 1; i < 6; i++ {
		w.addPkt(time.Now(), &Packet{
			SeqNo:   uint32(i),
			AckNo:   0,
			Flag:    ACK,
			Length:  1,
			Payload: []byte{'a'},
		})

		log.WithFields(log.Fields{
			"order": w.order,
			"pkts":  w.pkts,
			"new":   i,
		}).Info("addPkt")
	}

	w.addPkt(time.Now(), &Packet{
		SeqNo:   uint32(0),
		AckNo:   0,
		Flag:    ACK,
		Length:  1,
		Payload: []byte{'a'},
	})
	log.WithFields(log.Fields{
		"order": w.order,
		"pkts":  w.pkts,
		"new":   0,
	}).Info("addPkt")

	ack, err := w.cumAck(0)
	if err != nil {
		t.Error(err)
		return
	}

	log.WithFields(log.Fields{
		"order": w.order,
		"pkts":  w.pkts,
	}).Info("after cumAck")

	if ack != 6 {
		t.Errorf("expected cumAck 6, got %d", ack)
		return
	}

	s, err := w.start()
	if err == nil {
		t.Errorf("expected error, got %v", s)
		return
	}
}
