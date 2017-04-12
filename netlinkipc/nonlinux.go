// +build !linux

package netlinkipc

import (
	"fmt"

	"github.com/mdlayher/netlink"
)

func nlInit() (*netlink.Conn, error) {
	return nil, fmt.Errorf("netlink only supported on linux")
}
