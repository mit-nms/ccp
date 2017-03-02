package mmap

import (
	"os"

	log "github.com/Sirupsen/logrus"
	goMmap "github.com/edsrzf/mmap-go"
)

// mmap maps the given file into memory.
func Mmap(file string) (goMmap.MMap, error) {
	f, err := os.OpenFile(file, os.O_RDWR, 0777)
	if err != nil {
		log.Errorf("err opening file: %v", err)
		return goMmap.MMap([]byte{}), err
	}

	return MmapFile(f)
}

func MmapFile(file *os.File) (goMmap.MMap, error) {
	mm, err := goMmap.Map(file, goMmap.RDWR, 0)
	if err != nil {
		log.Error(err, file, goMmap.RDWR)
		return goMmap.MMap([]byte{}), err
	}

	return mm, nil
}
