// +build !linux

package main

import "fmt"

func main() {
	fmt.Println("netlink only supported on linux")
}
