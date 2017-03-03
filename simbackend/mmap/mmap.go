package mmap

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	goMmap "github.com/edsrzf/mmap-go"
	"zombiezen.com/go/capnproto2"
)

// use mmap'ed file as ring buffer
type mbuf struct {
	buf  goMmap.MMap
	rpos uint32
	wpos uint32
}

type MM struct {
	Enc *capnp.Encoder
	Dec *capnp.Decoder
	f   *os.File
	mm  *mbuf
}

// mmap maps the given file into memory.
func Mmap(file string) (MM, error) {
	f, err := os.OpenFile(file, os.O_RDWR, 0777)
	if err != nil {
		log.Errorf("err opening file: %v", err)
		return MM{}, err
	}

	mm, err := MmapFile(f)
	if err != nil {
		log.Errorf("err opening file: %v", err)
		return MM{}, err
	}

	return mm, nil
}

func MmapFile(file *os.File) (MM, error) {
	mbuf, err := mmap(file)
	if err != nil {
		return MM{}, err
	}

	return MM{
		Enc: capnp.NewEncoder(mbuf),
		Dec: capnp.NewDecoder(mbuf),
		f:   file,
		mm:  mbuf,
	}, nil
}

func mmap(file *os.File) (*mbuf, error) {
	mm, err := goMmap.Map(file, goMmap.RDWR, 0)
	if err != nil {
		log.Error(err, file, goMmap.RDWR)
		return nil, err
	}

	mbuf := &mbuf{buf: mm}

	return mbuf, nil
}

// implement io.Reader
func (m *mbuf) Read(p []byte) (n int, err error) {
	ring := append(m.buf[m.rpos:], m.buf[:m.rpos]...)

	n = copy(p, ring)
	m.rpos += uint32(n) % uint32(len(m.buf))

	log.WithFields(log.Fields{
		"copied": n,
		"wpos":   m.wpos,
		"rpos":   m.rpos,
		"read":   p,
		"buf":    m.buf[:m.rpos],
	}).Info("read from mmap")

	return n, nil
}

// implement io.Writer
func (m *mbuf) Write(p []byte) (n int, err error) {
	if len(p) > len(m.buf) {
		return 0, fmt.Errorf("write to mmap too big: %d > %d", len(p), len(m.buf))
	}

	n = copy(m.buf[m.wpos:], p)
	m.wpos += uint32(n) % uint32(len(m.buf))
	if n < len(p) {
		// wrapped around, copy rest
		n += copy(m.buf[m.wpos:], p[n:])
		if n != len(p) {
			return n, fmt.Errorf("wraparound error: %d < %d, wpos: %d", n, len(p), m.wpos)
		}
	}

	log.WithFields(log.Fields{
		"writeReq": len(p),
		"copied":   n,
		"wpos":     m.wpos,
		"rpos":     m.rpos,
		"write":    m.buf[:m.wpos],
	}).Info("write to mmap")

	return n, nil
}

func (m MM) Close() {
	m.mm.buf.Unmap()
	m.f.Close()
	os.Remove(m.f.Name())
	m.Enc = nil
	m.Dec = nil
}
