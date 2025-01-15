package main

import (
	"fmt"
	"net"
	"time"
)

const ServerIP string = "localhost" // "10.100.23.204"
const Port string = "34933"
const LocalPort string = "20011"

func UDPListenToServer(address string) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening on UDP:", err)
		return
	}
	defer conn.Close()

	for {
		buffer := make([]byte, 1024)
		n_bytes, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
		}
		fmt.Println("Received message:", string(buffer[0:n_bytes]))
	}
}

func UDPWriteToServer(address string) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Error dialing UDP:", err)
		return
	}
	defer conn.Close()

	for {
		_, err = conn.Write([]byte("Group54"))
		if err != nil {
			fmt.Println("Error writing to UDP:", err)
			continue
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func TCPClient() {
	address := fmt.Sprintf("%s:%s", ServerIP, Port)
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		fmt.Println("Error resolving TCP address:", err)
		return
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		fmt.Println("Error dialing TCP:", err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	_, err = conn.Write([]byte("Hello TCP"))
	if err != nil {
		fmt.Println("Error writing to TCP:", err)
	}

	data, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from TCP:", err)
		return
	}
	fmt.Println("Received message:", string(buffer[0:data]))
}

func requestConnectionFromServer() {
	address := fmt.Sprintf("%s:%s", ServerIP, Port)
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		fmt.Println("Error resolving TCP address:", err)
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		fmt.Println("Error dialing TCP:", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte("Connect to: 10.100.23.21:20011\x00"))
	if err != nil {
		fmt.Println("Error writing to TCP:", err)
	}
}

func handleClientConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from client:", err)
		return
	}
	fmt.Println("Received message from client:", string(buffer))

	_, err = conn.Write([]byte("Test"))
	if err != nil {
		fmt.Println("Error writing to client:", err)
	}
}

func TCPServer() {
	localAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%s", LocalPort))
	if err != nil {
		fmt.Println("Error resolving TCP address:", err)
		return
	}

	listener, err := net.ListenTCP("tcp", localAddr)
	if err != nil {
		fmt.Println("Error listening on TCP:", err)
		return
	}
	defer listener.Close()

	requestConnectionFromServer()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting TCP connection:", err)
			continue
		}
		go handleClientConnection(conn)
	}
}

// conn, err := listener.Accept()

// if err != nil {
// 	fmt.Println(err)
// }

// defer conn.Close()

// _, err = conn.Write([]byte("Test"))

// if err != nil {
// 	fmt.Println(err)
// }
// buffer := make([]byte, 1024)

// _, err = conn.Read(buffer)

// if err != nil {
// 	fmt.Println(err)
// }

// fmt.Println("We received something! ", string(buffer[:]))
// }

func main() {
	TCPServer()
	// go UDPListenToServer(fmt.Sprintf(":%s", LocalPort))
	// go UDPWriteToServer(fmt.Sprintf("%s:%s", ServerIP, LocalPort))

	// time.Sleep(100 * time.Second)
}
