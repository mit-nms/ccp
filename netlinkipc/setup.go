// +build linux

package netlinkipc

import (
	"golang.org/x/sys/unix"

	"github.com/mdlayher/netlink"
)

func nlInit() (*netlink.Conn, error) {
	nl, err := netlink.Dial(
		unix.NETLINK_USERSOCK,
		&netlink.Config{
			Groups: 0,
		},
	)
	if err != nil {
		return nil, err
	}

	nl.JoinGroup(NETLINK_MCAST_GROUP)

	return nl, nil
}
