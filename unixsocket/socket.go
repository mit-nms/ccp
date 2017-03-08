package unixsocket

import (
	"net"
	"os"

	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
	"zombiezen.com/go/capnproto2"
)

type SocketIpc struct {
	in  *net.UnixConn
	out *net.UnixConn

	ipc.BaseIpcLayer
}

func Setup() (ipc.IpcLayer, error) {
	addrIn, err := net.ResolveUnixAddr("unixgram", "/tmp/ccp-in")
	if err != nil {
		return nil, err
	}

	addrOut, err := net.ResolveUnixAddr("unixgram", "/tmp/ccp-out")
	if err != nil {
		return nil, err
	}

	in, err := net.ListenUnixgram("unixgram", addrOut)
	if err != nil {
		log.Error("listen error")
		return nil, err
	}

	out, err := net.DialUnix("unixgram", nil, addrIn)
	if err != nil {
		log.Error("dial error")
		return nil, err
	}

	s := &SocketIpc{
		in:  in,
		out: out,
	}

	go s.listen()
	return s, nil
}

func (s *SocketIpc) Close() error {
	s.in.Close()
	s.out.Close()
	os.Remove("/tmp/ccp-out")
	return nil
}

func (s *SocketIpc) SendAckMsg(socketId uint32, ackNo uint32) error {
	msg, err := ipc.MakeNotifyAckMsg(socketId, ackNo)
	if err != nil {
		return err
	}

	buf, err := msg.Marshal()
	if err != nil {
		return err
	}

	_, err = s.out.Write(buf)
	if err != nil {
		return err
	}

	return nil
}

func (s *SocketIpc) SendCwndMsg(socketId uint32, cwnd uint32) error {
	msg, err := ipc.MakeCwndMsg(socketId, cwnd)
	if err != nil {
		return err
	}

	buf, err := msg.Marshal()
	if err != nil {
		return err
	}

	_, err = s.out.Write(buf)
	if err != nil {
		return err
	}

	return nil
}

func (s *SocketIpc) listen() {
	dec := capnp.NewDecoder(s.in)
	for {
		msg, err := dec.Decode()
		if err != nil {
			continue
		}

		s.Parse(msg)
	}
}
