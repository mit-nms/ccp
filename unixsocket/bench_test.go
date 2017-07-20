package unixsocket

import (
	"bytes"
	"encoding/binary"
	"net"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
}

type TimeMsg struct {
	t time.Time
}

func (t TimeMsg) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, t.t.UnixNano())
	return buf.Bytes(), err
}

func BenchmarkUnixSocketRtt(b *testing.B) {
	reader, writer := setup()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go benchReader(reader)
		//benchWriter(writer)
		dur := benchWriter(writer)
		log.WithFields(log.Fields{
			"rtt": dur,
		}).Info("unix socket IPC RTT")
	}
}

func setup() (reader net.Conn, writer *SocketIpc) {
	ready := make(chan interface{})
	readerCh := make(chan net.Conn)
	os.RemoveAll("/tmp/ccp-test")
	go makeBenchReader(ready, readerCh)
	<-ready
	writer = makeBenchWriter()
	reader = <-readerCh
	return
}

func makeBenchReader(ready chan interface{}, ret chan net.Conn) {
	addr, err := net.ResolveUnixAddr("unixgram", "/tmp/ccp-test")
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warn("addr resolve error")
		return
	}

	in, err := net.ListenUnix("unix", addr)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warn("listen error")
		return
	}

	ready <- struct{}{}

	conn, err := in.AcceptUnix()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warn("accept error")
		return
	}

	ret <- conn
}

func makeBenchWriter() *SocketIpc {
	addr, err := net.ResolveUnixAddr("unix", "/tmp/ccp-test")
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warn("addr resolve error")
		return nil
	}

	out, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warn("dial error")
		return nil
	}

	return &SocketIpc{
		in:  nil,
		out: out,
	}
}

func benchReader(conn net.Conn) {
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warn("readfrom error")
		return
	}

	_, err = conn.Write(buf)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warn("writeto error")
		return
	}
}

func benchWriter(s *SocketIpc) time.Duration {
	err := s.SendMsg(&TimeMsg{t: time.Now()})
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warn("sendmsg error")
		return time.Hour
	}

	buf := make([]byte, 1024)
	_, err = s.out.Read(buf)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warn("read echo error")
		return time.Hour
	}

	var ns int64
	binary.Read(bytes.NewBuffer(buf), binary.LittleEndian, &ns)
	t := time.Unix(0, ns)

	return time.Since(t)
}
