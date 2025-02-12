package main

import (
	"Network/network/bcast"
	"Network/network/messages"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func genRandomInt(divisor int) int {
	i := rand.Int()
	if i%divisor != 0 {
		i--
	}
	return i
}

const (
	NEWHALLASSIGNMENTDIVISOR      int = 2
	HALLASSIGNMENTCOMPLETEDIVISOR     = 3
)

const PortNum int = 20011

var activeNodeIds []int

// Listens to incoming acknowledgment messages, distributes them to their corresponding channels
func IncomingAckDistributor(ackRx <-chan messages.Ack, HallAssignmentsAck chan<- messages.Ack) {

}

func HallAssignmentsTransmitter(HallAssignmentsTx chan<- messages.NewHallAssignments, OutgoingNewHallAssignments <-chan messages.NewHallAssignments) {
	currentAssignments := map[int]messages.NewHallAssignments{}

	timeOfAssignment := map[int]time.Time{}

	var newAssignment messages.NewHallAssignments
	for {
		select {
		case newAssignment = <-OutgoingNewHallAssignments:
			newAssignment.MessageID = genRandomInt(NEWHALLASSIGNMENTDIVISOR)

			currentAssignments[newAssignment.NodeID] = newAssignment
			timeOfAssignment[newAssignment.NodeID] = time.Now()
			HallAssignmentsTx <- newAssignment
		}
	}
}

func NetworkDude(id int) {

	AckTx := make(chan messages.Ack)
	AckRx := make(chan messages.Ack)

	ElevStatesTx := make(chan messages.ElevStates)
	ElevStatesRx := make(chan messages.ElevStates)

	HallAssignmentsTx := make(chan messages.NewHallAssignments)
	HallAssignmentsRx := make(chan messages.NewHallAssignments)

	go bcast.Transmitter(PortNum, AckTx, ElevStatesTx, HallAssignmentsTx)
	go bcast.Receiver(PortNum, AckRx, ElevStatesRx, HallAssignmentsRx)

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
	activeNodeIds = make([]int, 5)
	id, _ := strconv.Atoi(os.Args[1])

	go NetworkDude(id)
	for {
		time.Sleep(time.Second * 1000)
	}
}
