package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	adr, _ := net.ResolveUDPAddr("udp", ":20011")

	conn, err := net.DialUDP("udp", nil, adr)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Starting A Broadcast")
	msg := "I am alive!"
	i := 0
	for {
		time.Sleep(time.Millisecond * 500)

		conn.Write([]byte(msg))
		i += 1
		if i > 10 {
			break
		}
	}

	fmt.Println("Terminating A")
}
