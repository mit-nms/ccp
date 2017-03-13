package mmap

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"ccp/ipcBackend"

	"zombiezen.com/go/capnproto2"
)

type MMapIpc struct {
	in  MM
	out MM

	sockid   uint32
	listenCh chan *capnp.Message
	err      error
}

func New() ipcbackend.Backend {
	return &MMapIpc{}
}

func (m *MMapIpc) SetupListen(loc string, id uint32) ipcbackend.Backend {
	if m.err != nil {
		return m
	}

	var mm MM
	var err error
	if id != 0 {
		err = os.MkdirAll(fmt.Sprintf("/tmp/%d", id), 0755)
		if err != nil {
			m.err = err
			return m
		}

		mm, err = Mmap(fmt.Sprintf("/tmp/%d/%s", id, loc))
	} else {
		mm, err = Mmap(fmt.Sprintf("/tmp/%s", loc))
	}

	if err != nil {
		m.err = err
		return m
	}

	m.in = mm
	m.listenCh = make(chan *capnp.Message)
	go m.pollMmap()
	return m
}

func (m *MMapIpc) SetupSend(loc string, id uint32) ipcbackend.Backend {
	if m.err != nil {
		return m
	}

	var mm MM
	var err error
	if id != 0 {
		err = os.MkdirAll(fmt.Sprintf("/tmp/%d", id), 0755)
		if err != nil {
			m.err = err
			return m
		}

		mm, err = Mmap(fmt.Sprintf("/tmp/%d/%s", id, loc))
	} else {
		mm, err = Mmap(fmt.Sprintf("/tmp/%s", loc))
	}

	m.out = mm
	m.sockid = id
	return m
}

func (m *MMapIpc) SetupFinish() (ipcbackend.Backend, error) {
	if m.err != nil {
		return nil, m.err
	} else {
		return m, nil
	}
}

func Setup() (ipcbackend.Backend, error) {
	in, err := Mmap("/tmp/ccp-in")
	if err != nil {
		return nil, err
	}

	out, err := Mmap("/tmp/ccp-out")
	if err != nil {
		return nil, err
	}

	return &MMapIpc{
		in:       in,
		out:      out,
		listenCh: make(chan *capnp.Message),
	}, nil
}

func (m MMapIpc) Close() error {
	m.out.Close()
	os.RemoveAll(fmt.Sprintf("/tmp/%d", m.sockid))
	return nil
}

func (m *MMapIpc) SendMsg(msg *capnp.Message) error {
	return m.out.Enc.Encode(msg)
}

func (m *MMapIpc) ListenMsg() (chan *capnp.Message, error) {
	return m.listenCh, nil
}

func (m MMapIpc) pollMmap() {
	for _ = range time.Tick(time.Microsecond) {
		err := m.doDecode()
		if err != nil {
			continue
		}
	}
}

func (m *MMapIpc) doDecode() (err error) {
	// check first 64 bits. if 0s, no message
	first64 := m.in.mm.buf[:8]
	if bytes.Equal(
		[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		first64,
	) {
		return fmt.Errorf("empty buf")
	}

	msg, err := m.in.Dec.Decode()
	if err != nil {
		m.in.mm.reset()
		return
	}

	m.listenCh <- msg
	return
}
