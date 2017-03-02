package mmap

import (
	"os"
	"testing"
)

const FILENAME = "./mmap_test"

func setup(t *testing.T) {
	f, err := os.Create(FILENAME)
	if err != nil {
		t.Error(err)
	}

	f.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	f.Close()
}

func TestMmapRW(t *testing.T) {
	setup(t)

	mm, err := Mmap(FILENAME)
	if err != nil {
		t.Error(err)
	}

	wr := []byte("test\n")
	copy(mm, wr)

	mm.Flush()
	mm.Unmap()

	f, err := os.Open(FILENAME)
	if err != nil {
		t.Error(err)
	}

	b := make([]byte, 10)
	_, err = f.Read(b)
	if err != nil {
		t.Error(err)
	}

	expected := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	copy(expected, wr)

	if len(b) != len(expected) {
		t.Errorf("expected: %v\ngot: %v", expected, b)
	}

	for i := 0; i < len(b); i++ {
		if b[i] != expected[i] {
			t.Errorf("expected: %v\ngot: %v", expected, b)
		}
	}

	os.Remove(FILENAME)
}
