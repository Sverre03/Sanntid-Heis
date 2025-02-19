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
			if ackMsg.MessageID < IdPartitionSpace*int(NEW_HALL_ASSIGNMENT) {
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

func ElevStatesServer(commandCh <-chan string,
	TimeOfLastContactCh chan<- time.Time,
	elevStatesCh chan<- map[int]messages.ElevStates,
	activeElevIDsCh chan<- []int,
	elevStatesRx <-chan messages.ElevStates) {

	lastSeen := make(map[int]time.Time)
	knownNodes := make(map[int]messages.ElevStates)
	var TimeOfLastContact time.Time

	for {
		select {

		// if you get a msg on elevStatesRx:
		case elevState := <-elevStatesRx:
			id := elevState.NodeID
			// here, we must check if the id is ours. Placeholder for MyID is 0 for now.
			if id != 0 {

				// My new time of last contact
				TimeOfLastContact = time.Now()

				knownNodes[id] = elevState

				lastSeen[id] = time.Now()

			}

		case command := <-commandCh:
			switch command {
			case "getActiveElevStates":
				// remove dead connections before sending
				activeNodes := make(map[int]messages.ElevStates)
				for id, t := range lastSeen {
					if time.Since(t) < timeout {
						activeNodes[id] = knownNodes[id]
					}
				}
				// send the active nodes
				elevStatesCh <- activeNodes

			case "getActiveNodeIDs":
				activeIDs := make([]int, 0)

				for id, t := range lastSeen {
					if time.Since(t) < timeout {
						append(activeIDs, id)
					}
				}

				activeElevIDsCh <- activeIDs

			case "getTOLC":
				TimeOfLastContactCh <- TimeOfLastContact

			case "getAllKnownNodes":
				elevStatesCh <- knownNodes
			}
		}
	}
}

// func LightUpdateTransmitter(HallLightUpdateTx chan<- messages.HallLightUpdate,
// 	OutgoingLightUpdates chan messages.HallLightUpdate,
// 	HallLightUpdateAck <-chan messages.Ack) {

// 	activeAssignments := map[int]messages.HallLightUpdate{}

// 	timeoutChannel := make(chan int)

// 	var timedOutMsgID int
// 	var receivedAck messages.Ack
// 	var newAssignment messages.HallLightUpdate

// 	for {
// 		select {
// 		case newAssignment = <-OutgoingLightUpdates:

// 			// set a new message id
// 			newAssignment.MessageID = generateMessageID(HALL_LIGHT_UPDATE)

// 			// set/overwrite old assignments
// 			activeAssignments[newAssignment.NodeID] = newAssignment

// 			// send out the new assignment
// 			HallAssignmentsTx <- newAssignment

// 			// check for whether message is not acknowledged within duration
// 			time.AfterFunc(time.Millisecond*500, func() {
// 				timeoutChannel <- newAssignment.MessageID
// 			})

// 		case timedOutMsgID = <-timeoutChannel:

// 			// check if message is still in active assigments
// 			for _, msg := range activeAssignments {
// 				if msg.MessageID == timedOutMsgID {

// 					// add the message to the incoming messages channel
// 					OutgoingNewHallAssignments <- msg
// 					break
// 				}
// 			}

// 		case receivedAck = <-HallAssignmentsAck:

// 			// check if message is in map, if not do nothin
// 			if msg, ok := activeAssignments[receivedAck.NodeID]; ok {
// 				if msg.MessageID == receivedAck.MessageID {
// 					// remove the assignment from the map
// 					delete(activeAssignments, receivedAck.MessageID)
// 				}
// 			}
// 		}

// 	}
// }

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
