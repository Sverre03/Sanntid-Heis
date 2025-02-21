package main

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"time"
)

// note:
// this go file cannot be run from the VSCode terminal, it must be run in its own window (for some reason)

func main() {

	// address of udp broadcast
	adr, _ := net.ResolveUDPAddr("udp", ":20011")

	buffer := make([]byte, 1024)
	counter := 0

	// listener for the port, make conn obj
	connListener, err := net.ListenUDP("udp", adr)
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
		if err != nil {
			print(err)
		}

	}
	connListener.Close()

	fmt.Println("No primary on network, my time to shine")

	// start a writer to udp
	connWriter, err := net.DialUDP("udp", nil, adr)
	if err != nil {
		fmt.Println(err)
		return
	}

	// create a backup in new terminal
	err = exec.Command("gnome-terminal", "--", "go", "run", "C.go").Run()

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
