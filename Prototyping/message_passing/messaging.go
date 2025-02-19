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

const PortNum int = 20011
const IdPartitionSpace = 2 << 12
const timeout = 500 * time.Millisecond


type MessageIdPartition int

const (
	NEW_HALL_ASSIGNMENT MessageIdPartition = 0
	HALL_LIGHT_UPDATE                      = 1
	CONNECTION_REQ                         = 2
)

var activeNodeIds []int

func generateMessageID(partition MessageIdPartition) int {
	i := rand.Intn(IdPartitionSpace)
	i += (2 << 12) * int(partition)
	return i
}

// Listens to incoming acknowledgment messages, distributes them to their corresponding channels
func IncomingAckDistributor(ackRx <-chan messages.Ack, HallAssignmentsAck chan<- messages.Ack) {

	var ackMsg messages.Ack
	for {
		select {
		case ackMsg = <-ackRx:
			if ackMsg.MessageID < int(NEW_HALL_ASSIGNMENT) {
				HallAssignmentsAck <- ackMsg
			}
		}
	}
}

func HallAssignmentsTransmitter(HallAssignmentsTx chan<- messages.NewHallAssignments,
	OutgoingNewHallAssignments chan messages.NewHallAssignments,
	HallAssignmentsAck <-chan messages.Ack) {

	activeAssignments := map[int]messages.NewHallAssignments{}

	timeoutChannel := make(chan int)

	var timedOutMsgID int
	var receivedAck messages.Ack
	var newAssignment messages.NewHallAssignments

	for {
		select {
		case newAssignment = <-OutgoingNewHallAssignments:

			// set a new message id
			newAssignment.MessageID = generateMessageID(NEW_HALL_ASSIGNMENT)

			// set/overwrite old assignments
			activeAssignments[newAssignment.NodeID] = newAssignment

			// send out the new assignment
			HallAssignmentsTx <- newAssignment

			// check for whether message is not acknowledged within duration
			time.AfterFunc(time.Millisecond*500, func() {
				timeoutChannel <- newAssignment.MessageID
			})

		case timedOutMsgID = <-timeoutChannel:

			// check if message is still in active assigments
			for _, msg := range activeAssignments {
				if msg.MessageID == timedOutMsgID {

					// add the message to the incoming messages channel
					OutgoingNewHallAssignments <- msg
					break
				}
			}

		case receivedAck = <-HallAssignmentsAck:

			// check if message is in map, if not do nothin
			if msg, ok := activeAssignments[receivedAck.NodeID]; ok {
				if msg.MessageID == receivedAck.MessageID {
					// remove the assignment from the map
					delete(activeAssignments, receivedAck.MessageID)
				}
			}
		}

	}
}

func ElevStatesServer(commandChannel chan<- string, elevStates <-chan messages.ElevStates, , elevStatesRx <-chan messages.ElevStates) {
	lastSeen := make(map[int]time.Time)
	activeNodes = make(map[int]messages.ElevStates)

	for {
		select {
		case elevState := <- ElevStatesRx:
			
			if _, idExists := lastSeen[elevState.NodeID]; !idExists {
				map[elevState.NodeID] = elevState
			}

			lastSeen[id] = time.Now()
		

			// Removing dead connection
			for k, t := range lastSeen {
				if time.Since(t) > timeout {
					delete(lastSeen, k)
				}
			}
			
	
		}

		}
	}

}

func LightUpdateTransmitter(HallLightUpdateTx chan<- messages.HallLightUpdate,
	OutgoingLightUpdates chan messages.HallLightUpdate,
	HallLightUpdateAck <-chan messages.Ack) {

	activeAssignments := map[int]messages.HallLightUpdate{}

	timeoutChannel := make(chan int)

	var timedOutMsgID int
	var receivedAck messages.Ack
	var newAssignment messages.HallLightUpdate

	for {
		select {
		case newAssignment = <-OutgoingLightUpdates:

			// set a new message id
			newAssignment.MessageID = generateMessageID(HALL_LIGHT_UPDATE)

			// set/overwrite old assignments
			activeAssignments[newAssignment.NodeID] = newAssignment

			// send out the new assignment
			HallAssignmentsTx <- newAssignment

			// check for whether message is not acknowledged within duration
			time.AfterFunc(time.Millisecond*500, func() {
				timeoutChannel <- newAssignment.MessageID
			})

		case timedOutMsgID = <-timeoutChannel:

			// check if message is still in active assigments
			for _, msg := range activeAssignments {
				if msg.MessageID == timedOutMsgID {

					// add the message to the incoming messages channel
					OutgoingNewHallAssignments <- msg
					break
				}
			}

		case receivedAck = <-HallAssignmentsAck:

			// check if message is in map, if not do nothin
			if msg, ok := activeAssignments[receivedAck.NodeID]; ok {
				if msg.MessageID == receivedAck.MessageID {
					// remove the assignment from the map
					delete(activeAssignments, receivedAck.MessageID)
				}
			}
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

	CabRequestInfoTx := make(chan messages.CabRequestINF)
	CabRequestInfoRx := make(chan messages.CabRequestINF)

	GlobalHallRequestTx := make(chan messages.GlobalHallRequest)
	GlobalHallRequestRx := make(chan messages.GlobalHallRequest)

	HallLightUpdateTx := make(chan messages.HallLightUpdate)
	HallLightUpdateRx := make(chan messages.HallLightUpdate)

	ConnectionReqTx := make(chan messages.ConnectionReq)
	ConnectionReqRx := make(chan messages.ConnectionReq)

	NewHallReqTx := make(chan messages.NewHallRequest)
	NewHallReqRx := make(chan messages.NewHallRequest)

	HallAssignmentCompleteTx := make(chan messages.HallAssignmentComplete)
	HallAssignmentCompleteRx := make(chan messages.HallAssignmentComplete)

	go bcast.Transmitter(PortNum, AckTx, ElevStatesTx, HallAssignmentsTx, CabRequestInfoTx, GlobalHallRequestTx, HallLightUpdateTx, ConnectionReqTx, NewHallReqTx, HallAssignmentCompleteTx)
	go bcast.Receiver(PortNum, AckRx, ElevStatesRx, HallAssignmentsRx, CabRequestInfoRx, GlobalHallRequestRx, HallLightUpdateRx, ConnectionReqRx, NewHallReqRx, HallAssignmentCompleteRx)

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
