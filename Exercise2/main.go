package main

import (
	"fmt"
	"net"
	"time"
)

const bufferSize = 1024

const ServerIP string = "localhost" // "10.100.23.204"
const Port string = "34933"
const LocalPort string = "20011"
const WritePort string = "20000" // "20011" // Choose 20011 for both write and read when not working from home
const ReadPort string = "20001"  // "20011" // Choose 20011 when not working from home

// UDPListenToServer takes a port and listens to udp messages on it
// It prints anything it hears to the terminal
// If any error occurs, it returns without doing anything, printing an error message
func UDPListenToServer(port string) {
	addr, err := net.ResolveUDPAddr("udp", port)
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
		buffer := make([]byte, bufferSize)
		n_bytes, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			return
		}
		fmt.Println("Received message from server:", string(buffer[0:n_bytes]))
	}
}

// UDPWriteToServer takes an address and message and tries to send the message to it over udp
// if any error occurs, it returns without doing anything
func UDPWriteToServer(address, message string) {
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
		_, err = conn.Write([]byte(message))
		if err != nil {
			fmt.Println("Error writing to UDP:", err)
			continue
		}
		time.Sleep(1000 * time.Millisecond)
	}
}

// TCPCLient creates a TCP client that tries to connect to the given ip and port
// if the connection fails, it will return, printing an error message
// if the connection was successful, it will terminate the connection as soon as it receives a message
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

	buffer := make([]byte, bufferSize)

	data, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from TCP:", err)
		return
	}
	fmt.Println("Received welcome message:", string(buffer[0:data]))

	go func() {
		for {
			data, err = conn.Read(buffer)
			if err != nil {
				fmt.Println("Error reading from TCP:", err)
				return
			}

			fmt.Println("Received message:", string(buffer[0:data]))
		}
	}()

	for {
		_, err = conn.Write([]byte("Hello TCP"))
		if err != nil {
			fmt.Println("Error writing to TCP:", err)
			return
		}
		time.Sleep(1 * time.Second)
	}
}

// sends a fixed message to the sanntid server, requesting that it connects back to you
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

// takes a conn object, listens and sends a message
func handleClientConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, bufferSize)

	_, err := conn.Write([]byte("Welcome to TCP Server!\x00"))
	if err != nil {
		fmt.Println("Error writing to client:", err)
		return
	}

	for {
		_, err = conn.Read(buffer)
		if err != nil {
			fmt.Println("Error reading from client:", err)
			return
		}
		fmt.Println("Received message from client:", string(buffer))

		_, err = conn.Write([]byte("Hello from TCP Server!\x00"))
		if err != nil {
			fmt.Println("Error writing to client:", err)
			return
		}
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

func main() {
	// TCPServer()
	// go TCPServer()
	// time.Sleep(1 * time.Second)
	// TCPClient()
	// go UDPListenToServer(fmt.Sprintf("%s:%s", ServerIP, ReadPort))
	// go UDPWriteToServer(fmt.Sprintf("%s:%s", ServerIP, WritePort), "Hello UDP")

	// time.Sleep(100 * time.Second)
}
