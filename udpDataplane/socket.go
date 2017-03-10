package udpDataplane

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"ccp/ipc"

	log "github.com/Sirupsen/logrus"
	"github.mit.edu/hari/nimbus-cc/packetops"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

type Sock struct {
	name string

	conn        *net.UDPConn // the underlying connection
	writeBuf    []byte       // TODO make it a ring buffer
	writeBufPos int
	readBuf     []byte // TODO make it a ring buffer

	// sender
	cwnd           uint32
	lastAckedSeqNo uint32
	dupAckCnt      uint8
	nextSeqNo      uint32
	inFlight       window

	// receiver
	lastAck   uint32
	rcvWindow window

	// communication with CCP
	notifiedAckNo   uint32
	ackNotifyThresh uint32
	ipc             *ipc.Ipc

	// synchronization
	shouldTx   chan interface{}
	shouldPass chan uint32
	ackedData  chan uint32
	closed     chan interface{}

	mux sync.Mutex
}

// create and connect a socket.
// if ip == "", will listen on port until a SYN arrives
// else, will send a SYN to ip:port to establish a connection
func Socket(ip string, port string, name string) (*Sock, error) {
	log.WithFields(log.Fields{
		"ip":   ip,
		"port": port,
	}).Info("creating socket...")

	conn, err := makeConn(ip, port)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"ip":   ip,
		"port": port,
		"name": name,
	}).Info("created socket!")

	return mkSocket(conn, name)
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
		notifiedAckNo:   0,
		ackNotifyThresh: PACKET_SIZE * 10, // ~ 10 pkts
		// ipc initialized later

		// synchronization
		shouldTx:   make(chan interface{}, 1),
		shouldPass: make(chan uint32, 1),
		ackedData:  make(chan uint32),
		closed:     make(chan interface{}),
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

	err = s.setupIpc(uint32(lport))
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

func makeConn(ip string, port string) (conn *net.UDPConn, err error) {
	var addr *net.UDPAddr

	if ip != "" {
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
	} else {
		conn, addr, err = packetops.SetupListeningSock(port)
		if err != nil {
			return nil, err
		}

		log.Info("Listening for SYN")
		syn := &Packet{}
		conn, err = packetops.ListenForSyn(conn, addr, syn)
		if err != nil {
			return nil, err
		}

		syn.Flag = SYNACK
		syn.AckNo = 1

		log.Info("Sending SYNACK")
		err := packetops.SendSyn(conn, syn)
		if err != nil {
			return nil, err
		}
	}

	return
}

// currently only supports writing once
func (sock *Sock) Write(b []byte) (chan uint32, error) {
	if len(b) > len(sock.writeBuf) {
		return sock.ackedData, fmt.Errorf("Write exceeded buffer: %d > %d", len(b), len(sock.writeBuf))
	}

	if sock.checkClosed() {
		return nil, fmt.Errorf("closed socket")
	}

	sock.mux.Lock()
	defer sock.mux.Unlock()

	copy(sock.writeBuf, b)
	sock.writeBufPos = len(b)
	sock.shouldTx <- struct{}{}
	return sock.ackedData, nil
}

// receiver
func (sock *Sock) Read(returnGranularity uint32) chan []byte {
	passUp := make(chan []byte)
	go func() {
		notifiedDataNo := uint32(0)
		for lastAck := range sock.shouldPass {
			log.WithFields(log.Fields{
				"name":           sock.name,
				"sock.lastAck":   lastAck,
				"notifiedDataNo": notifiedDataNo,
			}).Debug("got send on shouldPass")

			if lastAck-notifiedDataNo > returnGranularity {
				select {
				case passUp <- sock.readBuf[notifiedDataNo:lastAck]:
					notifiedDataNo = lastAck
				default:
					log.Debug("skipping passUp ch notif")
				}

				log.WithFields(log.Fields{
					"name":       sock.name,
					"ackNo":      lastAck,
					"notifiedNo": notifiedDataNo,
				}).Info("got data")
			}
		}

		close(passUp)
	}()

	return passUp
}

func (sock *Sock) checkClosed() bool {
	return sock.conn == nil
}

func (sock *Sock) Fin() error {
	pkt := &Packet{
		SeqNo:   sock.nextSeqNo,
		AckNo:   sock.lastAck,
		Flag:    FIN,
		Length:  0,
		Payload: []byte{},
	}
	packetops.SendPacket(sock.conn, pkt, 0)
	return sock.Close()
}

func (sock *Sock) Close() error {
	log.WithFields(log.Fields{
		"name": sock.name,
	}).Info("closing")

	sock.mux.Lock()
	defer sock.mux.Unlock()
	err := sock.conn.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"name":  sock.name,
			"where": "closing",
		}).Warn(err)
	}

	sock.ipc.Close()
	close(sock.closed)
	return nil
}
