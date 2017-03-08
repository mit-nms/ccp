package mmap

import (
	"bytes"
	"fmt"
	"time"

	"ccp/ipcBackend"

	"zombiezen.com/go/capnproto2"
)

type MMapIpc struct {
	in  MM
	out MM

	listenCh chan *capnp.Message
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
	m.in.Close()
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
