package udpDataplane

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"ccp/ipc"

	"github.com/akshayknarayan/udp/packetops"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.InfoLevel)
}

type Sock struct {
	name string
	port uint32

	conn        *net.UDPConn // the underlying connection
	writeBuf    []byte       // TODO make it a ring buffer
	writeBufPos int
	readBuf     []byte // TODO make it a ring buffer

	// sender
	cwnd           uint32
	lastAckedSeqNo uint32
	dupAckCnt      uint8
	nextSeqNo      uint32
	inFlight       *window

	// receiver
	lastAck   uint32
	rcvWindow *window

	// communication with CCP
	ackNotifyThresh uint32
	ipc             *ipc.Ipc

	// synchronization
	shouldTx    chan interface{}
	shouldPass  chan uint32
	notifyAcks  chan notifyAck
	notifyDrops chan notifyDrop
	ackedData   chan uint32
	closed      chan interface{}

	mux sync.Mutex
}

// create and connect a socket.
// if ip == "", will listen on port until a SYN arrives
// else, will send a SYN to ip:port to establish a connection
func Socket(ip string, port string, name string) (s *Sock, err error) {
	s = <-socketNonBlocking(ip, port, name)
	if s == nil {
		err = fmt.Errorf("could not create socket")
	}

	return
}

func socketNonBlocking(ip string, port string, name string) (ch chan *Sock) {
	log.WithFields(log.Fields{
		"ip":   ip,
		"port": port,
	}).Info("creating socket...")

	ch = make(chan *Sock)
	if ip != "" {
		conn, err := makeConnClient(ip, port)
		if err != nil {
			log.WithFields(log.Fields{
				"ip":   ip,
				"port": port,
				"name": name,
				"err":  err,
			}).Warn("makeConnClient failed")
			goto fail
		}

		sk, err := mkSocket(conn, name)
		if err != nil {
			log.WithFields(log.Fields{
				"ip":   ip,
				"port": port,
				"name": name,
				"err":  err,
			}).Warn("mkSocket failed")
			goto fail
		}

		log.WithFields(log.Fields{
			"ip":   ip,
			"port": port,
			"name": name,
		}).Info("created socket!")

		go func() { ch <- sk }()
		return
	} else {
		connCh, err := makeConnServer(ip, port)
		if err != nil {
			log.WithFields(log.Fields{
				"ip":   ip,
				"port": port,
				"name": name,
				"err":  err,
			}).Warn("makeConnServer failed")
			goto fail
		}

		go func() {
			conn := <-connCh
			if conn == nil {
				ch <- nil
				return
			}

			log.WithFields(log.Fields{
				"ip":   ip,
				"port": port,
				"name": name,
			}).Info("created socket!")

			sk, err := mkSocket(conn, name)
			if err != nil {
				log.WithFields(log.Fields{
					"ip":   ip,
					"port": port,
					"name": name,
					"err":  err,
				}).Warn("mkSocket failed")
				ch <- nil
				return
			}

			ch <- sk
		}()
		return
	}

fail:
	go func() {
		ch <- nil
	}()
	return ch
}

func makeConnServer(ip string, port string) (connCh chan *net.UDPConn, err error) {
	conn, addr, err := packetops.SetupListeningSock(port)
	if err != nil {
		return nil, err
	}

	connCh = make(chan *net.UDPConn)

	go func() {
		log.Info("Listening for SYN")
		syn := &Packet{}
		conn, err := packetops.ListenForSyn(conn, addr, syn)
		if err != nil {
			log.WithFields(log.Fields{
				"ip":   ip,
				"port": port,
				"err":  err,
			}).Warn("makeConnServer failed")
			connCh <- nil
			return
		}

		syn.Flag = SYNACK
		syn.AckNo = 1

		log.Info("Sending SYNACK")
		err = packetops.SendSyn(conn, syn)
		if err != nil {
			log.WithFields(log.Fields{
				"ip":   ip,
				"port": port,
				"err":  err,
			}).Warn("makeConnServer failed")
			connCh <- nil
			return
		}

		connCh <- conn
	}()

	return
}

func makeConnClient(ip string, port string) (conn *net.UDPConn, err error) {
	var addr *net.UDPAddr
	conn, addr, err = packetops.SetupClientSock(ip, port)
	if err != nil {
		return nil, err
	}

	syn := &Packet{
		SeqNo: 0,
		AckNo: 0,
		Flag:  SYN,
	}

	log.Info("Sending SYN and expecting ACK")
	packetops.SynAckExchange(conn, addr, syn)
	return
}

func mkSocket(conn *net.UDPConn, name string) (*Sock, error) {
	s := &Sock{
		name: name,

		conn:     conn,
		writeBuf: make([]byte, 2e6),
		readBuf:  make([]byte, 2e6),

		//sender
		cwnd:           5 * PACKET_SIZE, // init_cwnd = ~ 1 pkt
		lastAckedSeqNo: 0,
		dupAckCnt:      0,
		nextSeqNo:      0,
		inFlight:       makeWindow(),

		//receiver
		lastAck:   0,
		rcvWindow: makeWindow(),

		// ccp communication
		ackNotifyThresh: PACKET_SIZE * 10, // ~ 10 pkts
		// ipc initialized later

		// synchronization
		shouldTx:    make(chan interface{}, 1),
		shouldPass:  make(chan uint32, 1),
		notifyAcks:  make(chan notifyAck, 1),
		notifyDrops: make(chan notifyDrop, 1),
		ackedData:   make(chan uint32),
		closed:      make(chan interface{}),
	}

	addr := s.conn.LocalAddr().String()
	spl := strings.Split(addr, ":")
	lport, err := strconv.Atoi(spl[1])
	if err != nil {
		log.WithFields(log.Fields{
			"name":  s.name,
			"addr":  spl,
			"lport": lport,
		}).Warn(err)
		lport = 42424
	}

	s.port = uint32(lport)
	err = s.setupIpc()
	if err != nil {
		log.WithFields(log.Fields{
			"name": s.name,
		}).Error(err)
		return nil, err
	}

	go s.rx()
	go s.tx()

	return s, nil
}

// currently only supports writing once
func (sock *Sock) Write(b []byte) (chan uint32, error) {
	if len(b) > len(sock.writeBuf) {
		return sock.ackedData, fmt.Errorf("Write exceeded buffer: %d > %d", len(b), len(sock.writeBuf))
	}

	sock.mux.Lock()
	copy(sock.writeBuf, b)
	sock.writeBufPos = len(b)
	sock.mux.Unlock()
	select {
	case sock.shouldTx <- struct{}{}:
	case <-sock.closed:
		return nil, fmt.Errorf("socket closed")
	}
	return sock.ackedData, nil
}

// receiver
func (sock *Sock) Read(returnGranularity uint32) chan []byte {
	passUp := make(chan []byte, 1)
	go func() {
		totAck := uint32(0)
		notifiedDataNo := uint32(0)
		timeout := time.NewTimer(time.Second)
	loop:
		for {
			select {
			case lastAck, ok := <-sock.shouldPass:
				if !ok {
					break loop
				}

				log.WithFields(log.Fields{
					"name":           sock.name,
					"acked":          totAck,
					"notifiedDataNo": notifiedDataNo,
				}).Debug("got send on shouldPass")

				if lastAck > totAck {
					totAck = lastAck
				}

				if !timeout.Stop() {
					<-timeout.C
				}
				timeout.Reset(time.Second)
			case <-timeout.C:
				timeout.Reset(time.Second)
			}

			if totAck-notifiedDataNo > returnGranularity {
				select {
				case passUp <- sock.readBuf[notifiedDataNo:totAck]:
					notifiedDataNo = totAck
					log.Debug("notified application of rcvd data")
				default:
				}
			}
		}

		close(passUp)
	}()

	return passUp
}

func (sock *Sock) Fin() error {
	sock.mux.Lock()
	pkt := &Packet{
		SeqNo:   sock.nextSeqNo,
		AckNo:   sock.lastAck,
		Flag:    FIN,
		Length:  0,
		Payload: []byte{},
	}
	sock.mux.Unlock()
	packetops.SendPacket(sock.conn, pkt, 0)
	return sock.Close()
}

func (sock *Sock) Close() error {
	log.WithFields(log.Fields{
		"name": sock.name,
	}).Info("closing")

	close(sock.closed)
	err := sock.conn.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"name":  sock.name,
			"where": "closing",
		}).Warn(err)
	}

	sock.ipc.Close()
	return nil
}
