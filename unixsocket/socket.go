package unixsocket

import (
	"net"
	"os"

	"ccp/ipcBackend"

	log "github.com/Sirupsen/logrus"
	"zombiezen.com/go/capnproto2"
)

type SocketIpc struct {
	in  *net.UnixConn
	out *net.UnixConn

	openFiles []string
	listenCh  chan *capnp.Message
	err       error
}

func New() ipcbackend.Backend {
	return &SocketIpc{openFiles: make([]string, 0)}
}

func (s *SocketIpc) SetupListen(loc string, id uint32) ipcbackend.Backend {
	if s.err != nil {
		return s
	}

	var addr *net.UnixAddr
	fd, newOpen, err := ipcbackend.AddressForListen(loc, id)
	if err != nil {
		s.err = err
		return s
	}

	s.openFiles = append(s.openFiles, newOpen)

	addr, err = net.ResolveUnixAddr("unixgram", fd)
	if err != nil {
		s.err = err
		return s
	}

	so, err := net.ListenUnixgram("unixgram", addr)
	if err != nil {
		s.err = err
		return s
	}

	s.in = so
	s.listenCh = make(chan *capnp.Message)
	go s.listen()
	return s
}

func (s *SocketIpc) SetupSend(loc string, id uint32) ipcbackend.Backend {
	if s.err != nil {
		return s
	}

	fd := ipcbackend.AddressForSend(loc, id)
	addr, err := net.ResolveUnixAddr("unixgram", fd)
	if err != nil {
		s.err = err
		return s
	}

	out, err := net.DialUnix("unixgram", nil, addr)
	if err != nil {
		s.err = err
		return s
	}

	s.out = out
	return s
}

func (s *SocketIpc) SetupFinish() (ipcbackend.Backend, error) {
	if s.err != nil {
		log.WithFields(log.Fields{
			"err": s.err,
		}).Error("error setting up IPC")
		return s, s.err
	} else {
		return s, nil
	}
}

func (s *SocketIpc) Close() error {
	if s.in != nil {
		s.in.Close()
	}

	for _, f := range s.openFiles {
		os.RemoveAll(f)
	}

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
