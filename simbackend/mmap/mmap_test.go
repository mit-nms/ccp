package mmap

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	goMmap "github.com/edsrzf/mmap-go"
)

const (
	FILENAME = "./mmap_test"
	EMPTY    = "0000000000"
	WR       = "test\n"
)

func setup() error {
	f, err := os.Create(FILENAME)
	if err != nil {
		return err
	}

	f.Write([]byte(EMPTY))
	f.Close()
	return nil
}

func TestMmapRW(t *testing.T) {
	err := setup()
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(FILENAME)

	mm, f, err := Mmap(FILENAME)
	if err != nil {
		t.Error(err)
	}

	copy(mm, WR)

	mm.Flush()
	mm.Unmap()
	f.Close()

	f, err = os.Open(FILENAME)
	if err != nil {
		t.Error(err)
	}

	b := make([]byte, 10)
	_, err = f.Read(b)
	if err != nil {
		t.Error(err)
	}

	expected := bytes.Repeat([]byte{'0'}, 10)
	copy(expected, WR)

	if len(b) != len(expected) {
		t.Errorf("expected: %v\ngot: %v", expected, b)
	}

	for i := 0; i < len(b); i++ {
		if b[i] != expected[i] {
			t.Errorf("expected: %v\ngot: %v", expected, b)
		}
	}
}

func TestConcurrentMmapCommunication(t *testing.T) {
	err := setup()
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(FILENAME)

	done := make(chan error)
	wrote := make(chan error)
	ready := make(chan interface{})
	go writer(ready, wrote)
	go reader(ready, wrote, done)

	err = <-done
	if err != nil {
		t.Error(err)
	}
}

func reader(ready chan interface{}, wrote chan error, done chan error) {
	mm, f, err := Mmap(FILENAME)
	if err != nil {
		done <- err
		return
	}

	// before writer does its thing, expect nothing here
	expected := bytes.Repeat([]byte{'0'}, 10)
	for i := 0; i < len(mm); i++ {
		if mm[i] != expected[i] {
			done <- fmt.Errorf("expected: %v\ngot: %v", expected, mm)
			return
		}
	}

	ready <- struct{}{}
	copy(expected, []byte(WR))

	err = <-wrote
	if err != nil {
		done <- err
		return
	}

	// after, expect what the writer wrote
	for i := 0; i < len(mm); i++ {
		if mm[i] != expected[i] {
			done <- fmt.Errorf("expected: %v\ngot: %v", expected, mm)
			return
		}
	}

	done <- nil
	mm.Unmap()
	f.Close()
}

func writer(ready chan interface{}, wrote chan error) {
	mm, f, err := Mmap(FILENAME)
	if err != nil {
		wrote <- err
		return
	}

	<-ready

	copy(mm, []byte(WR))

	wrote <- nil
	mm.Unmap()
	f.Close()
}

func BenchmarkConcurrentMmap(b *testing.B) {
	err := setup()
	if err != nil {
		b.Error(err)
	}
	done := make(chan error)
	wrote := make(chan error)
	ready := make(chan interface{})

	b.StartTimer()

	go func() {
		mm, _, err := Mmap(FILENAME)
		if err != nil {
			done <- err
			return
		}

		for {
			writerBench(mm, ready, wrote)
		}
	}()
	go func() {
		mm, _, err := Mmap(FILENAME)
		if err != nil {
			done <- err
			return
		}

		for {
			readerBench(mm, ready, wrote, done)
		}
	}()

	for i := 0; i < b.N; i++ {
		err = <-done
		if err != nil {
			b.Error(err)
		}
	}

	b.StopTimer()
	os.Remove(FILENAME)
}

func readerBench(mm goMmap.MMap, ready chan interface{}, wrote chan error, done chan error) {
	// before writer does its thing, expect nothing here
	expected := bytes.Repeat([]byte{'0'}, 10)
	for i := 0; i < len(mm); i++ {
		if mm[i] != expected[i] {
			done <- fmt.Errorf("expected: %v\ngot: %v", expected, mm)
			return
		}
	}

	ready <- struct{}{}
	copy(expected, []byte(WR))

	err := <-wrote
	if err != nil {
		done <- err
		return
	}

	// after, expect what the writer wrote
	for i := 0; i < len(mm); i++ {
		if mm[i] != expected[i] {
			done <- fmt.Errorf("expected: %v\ngot: %v", expected, mm)
			return
		}
	}

	// write 0s back to clear
	copy(mm, []byte(EMPTY))

	done <- nil
}

func writerBench(mm goMmap.MMap, ready chan interface{}, wrote chan error) {
	<-ready

	copy(mm, []byte(WR))

	wrote <- nil
}
