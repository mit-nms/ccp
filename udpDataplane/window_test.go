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

	cumAcked, _, err := w.rcvdPkt(time.Now(), &Packet{
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

func TestSendSack(t *testing.T) {
	w := makeWindow()

	for i := 0; i < 8*1460; i += 1460 {
		if i == 2*1460 || i == 5*1460 {
			continue
		}

		w.addPkt(time.Now(), &Packet{
			SeqNo:   uint32(i),
			AckNo:   0,
			Flag:    ACK,
			Length:  1460,
			Payload: []byte{'a'},
		})

		log.WithFields(log.Fields{
			"order": w.order,
			"pkts":  w.pkts,
			"new":   i,
		}).Info("addPkt")
	}

	ack, err := w.cumAck(0)
	if err != nil {
		t.Error(err)
		return
	}

	log.WithFields(log.Fields{
		"order": w.order,
		"pkts":  w.pkts,
	}).Info("after cumAck")

	if ack != 2*1460 {
		t.Errorf("expected cumAck 2920, got %d", ack)
		return
	}

	sack := w.getSack(ack)
	expectSack := []bool{true, true, false, true, true, false, false, false, false, false, false, false, false, false, false, false}
	if !equal(sack, expectSack) {
		t.Errorf("sack doesn't match:\nexpect %v\ngot %v", expectSack, sack)
		return
	}
}

func TestRecvSack(t *testing.T) {
	w := makeWindow()

	for i := 0; i < 10*1460; i += 1460 {
		w.addPkt(time.Now(), &Packet{
			SeqNo:   uint32(i),
			AckNo:   0,
			Flag:    ACK,
			Length:  1460,
			Payload: []byte{'a'},
		})

		log.WithFields(log.Fields{
			"order": w.order,
			"pkts":  w.pkts,
			"new":   i,
		}).Info("addPkt")
	}

	cumAcked, _, err := w.rcvdPkt(time.Now(), &Packet{
		SeqNo:  0,
		AckNo:  2 * 1460,
		Flag:   ACK,
		Length: 0,
		Sack: []bool{
			true,  // 3  = 4380
			false, // 4  = 5840
			true,  // 5  = 7300
			true,  // 6  = 8760
			true,  // 7  = 10220
			false, // 8  = 11680
			false, // 9  = 13140
			true,  // 10 = 14600
			false, // 11 = 16060
			false, // 12 = 17520
			false, // 13 = 18980
			false, // 14 = 20440
			false, // 15 = 21900
			false, // 16 = 23360
			false, // 17 = 24820
			false, // 18 = 26280
		},
		Payload: []byte{},
	})
	if err != nil {
		t.Error(err)
		return
	}

	if cumAcked != 2*1460 {
		t.Errorf("wrong cumAck on sender\nexpected 2920\ngot %v", cumAcked)
		return
	}

	if len(w.order) != 4 {
		t.Errorf("wrong order\nexpected [2920 5840 11680 13140]\ngot %v", w.order)
		return
	}
	for i, v := range w.order {
		switch i {
		case 0:
			if v != 2920 {
				goto fail
			}
		case 1:
			if v != 5840 {
				goto fail
			}
		case 2:
			if v != 11680 {
				goto fail
			}
		case 3:
			if v != 13140 {
				goto fail
			}
		}
	}

	goto pass

fail:
	t.Errorf("wrong order\nexpected [2920 5840 11680 13140]\ngot %v", w.order)
pass:
	return

}
