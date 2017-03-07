package udpDataplane

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestBasicTransfer(t *testing.T) {
	fmt.Println("starting...")
	testTransfer(t, bytes.Repeat([]byte{'a'}, 100))
}

func testTransfer(t *testing.T, data []byte) {
	done := make(chan interface{})
	fmt.Println("starting receiver...")
	go receiver(t, data, done)
	<-time.After(time.Duration(100) * time.Millisecond)
	fmt.Println("starting sender...")
	go sender(t, data)
	<-done
}

func sender(t *testing.T, data []byte) {
	sock, err := Socket("127.0.0.1", "60000", "SENDER")
	if err != nil {
		t.Error(err)
		return
	}

	sock.Write(data)
}

func receiver(t *testing.T, expect []byte, done chan interface{}) {
	sock, err := Socket("", "60000", "RCVR")
	if err != nil {
		t.Error(err)
		return
	}

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
		t.Errorf("received data doesn't match: \n%v\n%v", buf, expect)
	}

	for i := 0; i < len(buf); i++ {
		if buf[i] != expect[i] {
			t.Errorf("received data doesn't match: \n%v\n%v", buf, expect)
		}
	}

	done <- struct{}{}
}
