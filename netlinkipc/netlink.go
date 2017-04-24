package netlinkipc

import (
	"ccp/ipcBackend"

	log "github.com/Sirupsen/logrus"
	"github.com/mdlayher/netlink"
)

const (
	NETLINK_MCAST_GROUP = 22
)

type NetlinkIpc struct {
	conn *netlink.Conn

	listenCh chan []byte

	err    error
	killed chan interface{}
}

func New() ipcbackend.Backend {
	return &NetlinkIpc{}
}

func (n *NetlinkIpc) SetupSend(loc string, id uint32) ipcbackend.Backend {
	if n.err != nil {
		return n
	}

	if n.conn == nil {
		nl, err := nlInit()
		if err != nil {
			n.err = err
			return n
		}

		n.conn = nl
	}

	return n
}

func (n *NetlinkIpc) SetupListen(loc string, id uint32) ipcbackend.Backend {
	if n.err != nil {
		return n
	}

	if n.conn == nil {
		nl, err := nlInit()
		if err != nil {
			n.err = err
			return n
		}

		n.conn = nl
	}

	n.listenCh = make(chan []byte)
	go n.listen()

	return n
}

func (s *NetlinkIpc) SetupFinish() (ipcbackend.Backend, error) {
	if s.err != nil {
		log.WithFields(log.Fields{
			"err": s.err,
		}).Error("error setting up IPC")
		return s, s.err
	} else {
		s.killed = make(chan interface{})
		return s, nil
	}
}

func (n *NetlinkIpc) GetCreateMsg() ipcbackend.CreateMsg {
	return &CreateMsg{}
}

func (n *NetlinkIpc) GetAckMsg() ipcbackend.AckMsg {
	return &AckMsg{}
}

func (n *NetlinkIpc) GetCwndMsg() ipcbackend.CwndMsg {
	return &CwndMsg{}
}

func (n *NetlinkIpc) GetDropMsg() ipcbackend.DropMsg {
	return &DropMsg{}
}

func (n *NetlinkIpc) SendMsg(msg ipcbackend.Msg) error {
	buf, err := msg.Serialize()
	if err != nil {
		return err
	}

	resp := netlink.Message{
		Header: netlink.Header{
			Flags: netlink.HeaderFlagsRequest | netlink.HeaderFlagsAcknowledge,
		},
		Data: buf,
	}

	_, err = n.conn.Send(resp)
	return err
}

func (n *NetlinkIpc) listen() {
	for {
		select {
		case <-n.killed:
			n.conn.Close()
			close(n.listenCh)
			return
		default:
		}

		msgs, err := n.conn.Receive()
		if err != nil {
			log.WithFields(log.Fields{
				"where": "netlinkipc.listen",
			}).Warn(err)
			return
		}

		for _, msg := range msgs {
			n.listenCh <- msg.Data
		}
	}
}

func (n *NetlinkIpc) Listen() chan ipcbackend.Msg {
	msgCh := make(chan ipcbackend.Msg)

	go func() {
		for {
			select {
			case <-n.killed:
				close(msgCh)
				return
			case buf := <-n.listenCh:
				m := parse(buf)
				msgCh <- m
			}
		}
	}()

	return msgCh
}

func (n *NetlinkIpc) Close() error {
	close(n.killed)
	return nil
}

func parse(buf []byte) (msg ipcbackend.Msg) {
	t, err := readType(buf)
	if err != nil {
		log.WithFields(log.Fields{
			"type": t,
		}).Panic(err)
	}

	switch t {
	case CREATE:
		msg = &CreateMsg{}
	case DROP:
		msg = &DropMsg{}
	case CWND:
		msg = &CwndMsg{}
	case ACK:
		msg = &AckMsg{}
	default:
		log.WithFields(log.Fields{
			"type": t,
		}).Panic("unknown type")
	}

	err = msg.Deserialize(buf)
	if err != nil {
		log.WithFields(log.Fields{
			"msg": msg,
		}).Panic(err)
	}
	return
}
