package udpDataplane

import (
	"bytes"
	"testing"

	"github.com/akshayknarayan/udp/packetops"
)

func equal(a []bool, b []bool) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestEncodePacket(t *testing.T) {
	p := &Packet{
		SeqNo:   0,
		AckNo:   0,
		Flag:    ACK,
		Length:  10,
		Sack:    []bool{},
		Payload: bytes.Repeat([]byte{'t'}, 10),
	}

	enc, err := p.Encode(0)
	if err != nil {
		t.Errorf("encoding error: %v", err)
	}

	expected := bytes.Repeat([]byte{0}, 8) // seq and ack
	expected = append(expected, []byte{0xa, 0x20}...)
	expected = append(expected, []byte{0x00, 0x00}...)
	expected = append(expected, bytes.Repeat([]byte{'t'}, 10)...)

	if len(enc.Buf) != len(expected) {
		t.Errorf("wrong length:\n%v\n%v", enc, expected)
		return
	}

	for i := 0; i < len(expected); i++ {
		if enc.Buf[i] != expected[i] {
			t.Errorf("expected %v\ngot %v", enc, expected)
			return
		}
	}
}

func TestDecodePacket(t *testing.T) {
	r := bytes.Repeat([]byte{0}, 8) // seq and ack
	r = append(r, []byte{0xa, 0x20}...)
	r = append(r, []byte{0x01, 0x00}...)
	r = append(r, bytes.Repeat([]byte{'t'}, 10)...)

	raw := &packetops.RawPacket{
		Buf:  r,
		From: nil,
	}

	expected := &Packet{
		SeqNo:   0,
		AckNo:   0,
		Flag:    ACK,
		Length:  10,
		Sack:    []bool{true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
		Payload: bytes.Repeat([]byte{'t'}, 10),
	}

	got := &Packet{}
	err := got.Decode(raw)
	if err != nil {
		t.Errorf("decoding error: %v", err)
	}

	if got.SeqNo != expected.SeqNo ||
		got.AckNo != expected.AckNo ||
		got.Flag != expected.Flag ||
		got.Length != expected.Length ||
		!equal(got.Sack, expected.Sack) {
		t.Errorf("expected: %v\ngot: %v", expected, got)
		return
	}

	for i := 0; i < int(got.Length); i++ {
		if got.Payload[i] != expected.Payload[i] {
			t.Errorf("expected %v\ngot %v", expected, got)
			return
		}
	}
}
