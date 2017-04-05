// +build linux

package main

import (
	"fmt"
	"golang.org/x/sys/unix"

	"github.com/mdlayher/netlink"
)

func main() {
	nl, err := netlink.Dial(unix.NETLINK_USERSOCK, &netlink.Config{Groups: 0})
	if err != nil {
		fmt.Println(err)
		return
	}

	nl.JoinGroup(uint32(22))

	fmt.Println("receiving...")
	msgs, err := nl.Receive()
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, msg := range msgs {
		fmt.Printf("got message: %s\n", string(msg.Data))
	}

	resp := netlink.Message{
		Header: netlink.Header{
			Flags: netlink.HeaderFlagsRequest | netlink.HeaderFlagsAcknowledge,
		},
		Data: []byte("response from go userspace"),
	}

	nl.Send(resp)
}
