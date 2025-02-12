package main

import (
	"Network/network/bcast"
	"Network/network/messages"
	"fmt"
	"os"
	"strconv"
	"time"
)

const PortNum int = 20011

func NetworkDude(id int) {

	AckTx := make(chan messages.Ack)
	AckRx := make(chan messages.Ack)

	ElevStatesTx := make(chan messages.ElevStates)
	ElevStatesRx := make(chan messages.ElevStates)

	NewHallAssignmentsTx := make(chan messages.NewHallAssignments)
	NewHallAssignmentsRx := make(chan messages.NewHallAssignments)

	go bcast.Transmitter(PortNum, AckTx, ElevStatesTx, NewHallAssignmentsTx)
	go bcast.Receiver(PortNum, AckRx, ElevStatesRx, NewHallAssignmentsRx)
	fmt.Println("Started the network dude")

	for {
		select {
		case states := <-ElevStatesRx:
			fmt.Println(states.NodeID)
			fmt.Println(states.Behavior)
		case <-time.After(time.Millisecond * 500):
			ElevStatesTx <- messages.ElevStates{id, "down", 3, [4]bool{false, false, false, false}, "Idle"}
		}
	}
}

func main() {
	id, _ := strconv.Atoi(os.Args[1])

	go NetworkDude(id)
	for {
		time.Sleep(time.Second * 1000)
	}
}
