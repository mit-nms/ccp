package unixsocket

import (
	"net"
	"os"

	"simbackend/ipc"

	log "github.com/Sirupsen/logrus"
	"zombiezen.com/go/capnproto2"
)

type SocketIpc struct {
	in  *net.UnixConn
	out *net.UnixConn
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

	return &SocketIpc{
		in:  in,
		out: out,
	}, nil
}

func (s *SocketIpc) Close() error {
	s.in.Close()
	s.out.Close()
	os.Remove("/tmp/ccp-out")
	return nil
}

func (s *SocketIpc) Send(socketId uint32, ackNo uint32) error {
	msg, err := ipc.NotifyAckMsg(socketId, ackNo)
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

func (s *SocketIpc) Listen() (chan uint32, error) {
	gotMsg := make(chan uint32)
	go s.listen(gotMsg)
	return gotMsg, nil
}

func (s *SocketIpc) listen(gotMsg chan uint32) {
	dec := capnp.NewDecoder(s.in)
	for {
		msg, err := dec.Decode()
		if err != nil {
			continue
		}

		cwnd, err := ipc.ReadCwndMsg(msg)
		if err != nil {
			continue
		}

		gotMsg <- cwnd
	}
}
