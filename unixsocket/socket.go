package unixsocket

import (
	"net"
	"os"

	"ccp/ipcBackend"

	log "github.com/sirupsen/logrus"
)

type SocketIpc struct {
	in  *net.UnixConn
	out *net.UnixConn

	openFiles []string

	listenCh chan []byte

	err    error
	killed chan interface{}
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
	s.killed = make(chan interface{})
	s.listenCh = make(chan []byte)
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

func (s *SocketIpc) SendMsg(msg ipcbackend.Msg) error {
	buf, err := msg.Serialize()
	if err != nil {
		return err
	}

	_, err = s.out.Write(buf)
	if err != nil {
		return err
	}

	return nil
}

func (s *SocketIpc) Listen() chan []byte {
	msgCh := make(chan []byte)

	go func() {
		for {
			select {
			case <-s.killed:
				close(msgCh)
				return
			case buf := <-s.listenCh:
				msgCh <- buf
			}
		}
	}()

	return msgCh
}

func (s *SocketIpc) Close() error {
	close(s.killed)
	return nil
}

func (s *SocketIpc) listen() {
	buf := make([]byte, 2048)
	writePos := 0
	for {
		select {
		case _, ok := <-s.killed:
			if !ok {
				if s.in != nil {
					s.in.Close()
				}

				for _, f := range s.openFiles {
					os.RemoveAll(f)
				}

				close(s.listenCh)

				log.WithFields(log.Fields{
					"where": "socketIpc.listenMsg.checkKilled",
				}).Info("killed, closing")
				return
			}
		default:
		}

		ring := append(buf[writePos:], buf[:writePos]...)
		n, err := s.in.Read(ring)
		if err != nil {
			log.WithFields(log.Fields{
				"where": "socketIpc.listenMsg.read",
			}).Warn(err)
			continue
		}

		writePos = (writePos + n) % len(buf)
		s.listenCh <- ring[:n]
	}
}
