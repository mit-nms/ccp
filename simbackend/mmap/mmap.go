package mmap

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	goMmap "github.com/edsrzf/mmap-go"
	"zombiezen.com/go/capnproto2"
)

type mbuf struct {
	buf goMmap.MMap
}

type MM struct {
	Enc *capnp.Encoder
	Dec *capnp.Decoder
	f   *os.File
	mm  mbuf
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

func mmap(file *os.File) (mbuf, error) {
	mm, err := goMmap.Map(file, goMmap.RDWR, 0)
	if err != nil {
		log.Error(err, file, goMmap.RDWR)
		return mbuf{}, err
	}

	mbuf := mbuf{buf: mm}

	return mbuf, nil
}

// implement io.Reader
func (m mbuf) Read(p []byte) (n int, err error) {
	if len(m.buf) > len(p) {
		return 0, fmt.Errorf("read from mmap too big: %d", len(p))
	}

	copy(p, m.buf)

	return len(p), nil
}

// implement io.Writer
func (m mbuf) Write(p []byte) (n int, err error) {
	if len(p) > len(m.buf) {
		return 0, fmt.Errorf("write to mmap too big: %d", len(p))
	}

	copy(m.buf, p)

	return len(p), nil
}

func (m MM) Close() {
	m.mm.buf.Unmap()
	m.f.Close()
	m.Enc = nil
	m.Dec = nil
}
