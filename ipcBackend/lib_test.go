package ipcbackend

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestListenAddr(t *testing.T) {
	f, o, err := AddressForListen("addr-test", 2)
	if err != nil {
		t.Error(err)
		return
	}

	defer func() {
		os.RemoveAll(o)
	}()

	if !bytes.Equal([]byte(f), []byte(fmt.Sprintf("/tmp/ccp-2/addr-test"))) {
		t.Error("wrong value", f)
		return
	}
}
