package main

import (
	"fmt"
	"net"
	"time"
)

// We define some custom struct to send over the network.
// Note that all members we want to transmit must be public. Any private members
//
//	will be received as zero-values.
type HelloMsg struct {
	Message string
	Iter    int
}

const ServerIP string = "10.100.23.204"
const Port int = 20011

func listenToServer(address string) {
	addr, err := net.ResolveUDPAddr("udp", address)

	if err != nil {
		fmt.Println("Connection failed listen")
	}

	conn, err := net.ListenUDP("udp", addr)

	if err != nil {
		fmt.Println("Connection failed listen")
	}

	for {
		buffer := make([]byte, 1024)
		n_bytes, _, err := conn.ReadFromUDP(buffer)

		if err != nil {
			fmt.Println("Connection failed listen")
		}
		fmt.Println(string(buffer[0:n_bytes]))
	}
}

func writeToServer(address string) {
	addr, err := net.ResolveUDPAddr("udp", address)

	if err != nil {
		fmt.Println("Connection failed write")
	}

	conn, err := net.DialUDP("udp", nil, addr)

	if err != nil {
		fmt.Println("Connection failed write")
	}

	for {

		_, err = conn.Write([]byte("Group54"))

		if err != nil {
			fmt.Println("Connection failed write")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func main() {
	go listenToServer(":30000")
	go writeToServer(fmt.Sprintf("%s:%d", ServerIP, Port))

	time.Sleep(100 * time.Second)
}
