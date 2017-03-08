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
	log.SetLevel(log.InfoLevel)
}

type Sock struct {
	name string

	conn        *net.UDPConn // the underlying connection
	writeBuf    []byte       // TODO make it a ring buffer
	writeBufPos int
	readBuf     []byte // TODO make it a ring buffer

	seqNo           uint32
	cumAck          uint32
	ackNo           uint32
	lastAckSent     uint32
	notifiedAckNo   uint32
	notifiedDataNo  uint32
	ackNotifyThresh uint32
	cwnd            uint32
	inFlight        uint32

	// communication with CCP
	ipc ipc.Ipc

	// synchronization
	shouldTx   chan interface{}
	shouldPass chan interface{}
	passUp     chan []byte

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

	s := &Sock{
		name:            name,
		conn:            conn,
		writeBuf:        make([]byte, 2e6),
		readBuf:         make([]byte, 2e6),
		seqNo:           0,
		cumAck:          0,
		ackNo:           0,
		lastAckSent:     0,
		notifiedAckNo:   0,
		notifiedDataNo:  0,
		ackNotifyThresh: PACKET_SIZE * 10, // ~ 10 pkts
		cwnd:            PACKET_SIZE,      // init_cwnd = ~ 1 pkt
		inFlight:        0,
		shouldTx:        make(chan interface{}, 1),
		shouldPass:      make(chan interface{}, 1),
		passUp:          make(chan []byte),
	}

	log.WithFields(log.Fields{
		"ip":   ip,
		"port": port,
		"name": s.name,
	}).Info("created socket!")

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
func (sock *Sock) Write(b []byte) error {
	if len(b) > len(sock.writeBuf) {
		return fmt.Errorf("Write exceeded buffer: %d > %d", len(b), len(sock.writeBuf))
	}

	copy(sock.writeBuf, b)
	sock.writeBufPos = len(b)

	sock.shouldTx <- struct{}{}

	return nil
}

func (sock *Sock) Read(returnGranularity uint32) chan []byte {
	sock.passUp = make(chan []byte)

	go func() {
		for _ = range sock.shouldPass {
			if sock.ackNo-sock.notifiedDataNo > returnGranularity {
				sock.passUp <- sock.readBuf[sock.notifiedDataNo:sock.ackNo]
				sock.notifiedDataNo = sock.ackNo

				log.WithFields(log.Fields{
					"name":       sock.name,
					"ackNo":      sock.ackNo,
					"notifiedNo": sock.notifiedDataNo,
				}).Info("got data")
			}
		}
	}()

	return sock.passUp
}

func (sock *Sock) checkClosed() bool {
	sock.mux.Lock()
	defer sock.mux.Unlock()

	return sock.conn == nil
}

func (sock *Sock) Close() error {
	log.WithFields(log.Fields{
		"name": sock.name,
	}).Info("closing")

	err := sock.conn.Close()
	if err != nil {
		return err
	}

	sock.conn = nil
	close(sock.passUp)

	sock.ipc.Close()
	return nil
}
