package unixsocket

import (
	"fmt"
	"net"
	"os"

	"ccp/ipcBackend"

	log "github.com/Sirupsen/logrus"
	"zombiezen.com/go/capnproto2"
)

type SocketIpc struct {
	in  *net.UnixConn
	out *net.UnixConn

	sockid uint32

	listenCh chan *capnp.Message
}

func SetupCcp() (ipcbackend.Backend, error) {
	addrIn, err := net.ResolveUnixAddr("unixgram", "/tmp/ccp-in")
	if err != nil {
		return nil, err
	}

	in, err := net.ListenUnixgram("unixgram", addrIn)
	if err != nil {
		log.Error("listen error")
		return nil, err
	}

	s := &SocketIpc{
		in:       in,
		listenCh: make(chan *capnp.Message),
	}

	go s.listen()
	return s, nil
}

func SetupClient(sockid uint32) (ipcbackend.Backend, error) {
	addrOut, err := net.ResolveUnixAddr("unixgram", "/tmp/ccp-in")
	if err != nil {
		return nil, err
	}

	out, err := net.DialUnix("unixgram", nil, addrOut)
	if err != nil {
		log.Error("dial error")
		return nil, err
	}

	err = os.MkdirAll(fmt.Sprintf("/tmp/%d", sockid), 0755)
	if err != nil {
		return nil, err
	}

	addrIn, err := net.ResolveUnixAddr("unixgram", fmt.Sprintf("/tmp/%d/ccp-out", sockid))
	if err != nil {
		return nil, err
	}

	in, err := net.ListenUnixgram("unixgram", addrIn)
	if err != nil {
		log.Error("listen error")
		return nil, err
	}

	s := &SocketIpc{
		in:       in,
		out:      out,
		sockid:   sockid,
		listenCh: make(chan *capnp.Message),
	}

	go s.listen()
	return s, nil
}

func (s *SocketIpc) Close() error {
	if s.in != nil {
		s.in.Close()
	}

	if s.out != nil {
		s.out.Close()
	}

	os.RemoveAll(fmt.Sprintf("/tmp/%d", s.sockid))
	return nil
}

func (s *SocketIpc) SendMsg(msg *capnp.Message) error {
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

func (s *SocketIpc) ListenMsg() (chan *capnp.Message, error) {
	return s.listenCh, nil
}

func (s *SocketIpc) listen() {
	dec := capnp.NewDecoder(s.in)
	for {
		msg, err := dec.Decode()
		if err != nil {
			continue
		}

		s.listenCh <- msg
	}
}
