package main

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

// note:
// this go file cannot be run from the VSCode terminal, it must be run in its own window (for some reason)

func main() {

	// address of udp broadcast
	listenAddress, _ := net.ResolveUDPAddr("udp", ":20011")
	broadcastAddress, _ := net.ResolveUDPAddr("udp", "255.255.255.255:20011")

	buffer := make([]byte, 1024)
	counter := 0

	// listener for the port, make conn obj
	connListener, err := net.ListenUDP("udp", listenAddress)
	if err != nil {
		fmt.Println(err)
		return
	}

	// start by listening for active program "counting" on local network
	for {
		// IMPORTANT: update the read deadline every time the loop loops
		connListener.SetReadDeadline(time.Now().Add(time.Millisecond * 1500))
		n, listenErr := connListener.Read(buffer)
		if listenErr != nil {
			// if the listener timed out
			//fmt.Println(err)
			break
		}

		counter, err = strconv.Atoi(string(buffer[0:n]))
		fmt.Printf("Received %d \n", counter)
		if err != nil {
			print(err)
		}

	}
	connListener.Close()

	fmt.Println("No primary on network, my time to shine")

	// start a writer to udp
	connWriter, err := net.DialUDP("udp", nil, broadcastAddress)
	if err != nil {
		fmt.Println(err)
		return
	}

	// create a new backup on other pc (does not work)
	//err = exec.Command("gnome-terminal", "--", "go", "run", "C.go").Run()
	//err = exec.Command("ssh", "-i", "id_rsa", "student@10.100.23.27", "DISPLAY=:0 gnome-terminal -- go run /home/student/Desktop/C.go").Run()

	if err != nil {
		fmt.Println(err)
		fmt.Println("Could not run the backup, do not run this prog in VSCode integrated terminal :(")
		return
	}

	for {

		// increment counter and update the send buffer
		counter += 1
		buffer = []byte(fmt.Sprintf("%d", counter))

		fmt.Println(fmt.Sprintf("%d", counter))

		// send the new counter value on the network
		connWriter.Write(buffer)

		time.Sleep(time.Millisecond * 500)
	}
}
