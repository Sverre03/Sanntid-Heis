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

// some of these constants may need to live elsewhere
const PortNum int = 20011
const IDPartitionSize = 2 << 12
const timeout = 500 * time.Millisecond
const MASTER_TRANSMIT_INTERVAL = 50 * time.Millisecond

type MessageIDPartition int

const (
	NEW_HALL_ASSIGNMENT      MessageIDPartition = 0
	HALL_LIGHT_UPDATE        MessageIDPartition = 1
	CONNECTION_REQ           MessageIDPartition = 2
	CAB_REQ_INFO             MessageIDPartition = 3
	HALL_ASSIGNMENT_COMPLETE MessageIDPartition = 4
)

// generates a message ID that corresponsds to the message type
func GenerateMessageID(partition MessageIDPartition) int {
	i := rand.Intn(IDPartitionSize)
	i += (2 << 12) * int(partition)
	return i
}

// Listens to incoming acknowledgment messages from UDP, distributes them to their corresponding channels
func IncomingAckDistributor(ackRx <-chan messages.Ack,
	hallAssignmentsAck chan<- messages.Ack,
	lightUpdateAck chan<- messages.Ack,
	connectionReqAck chan<- messages.Ack,
	cabReqInfoAck chan<- messages.Ack,
	hallAssignmentCompleteAck chan<- messages.Ack) {

	for ackMsg := range ackRx {

		if ackMsg.MessageID < IDPartitionSize*int(NEW_HALL_ASSIGNMENT) {
			hallAssignmentsAck <- ackMsg

		} else if ackMsg.MessageID < IDPartitionSize*int(HALL_LIGHT_UPDATE) {
			lightUpdateAck <- ackMsg

		} else if ackMsg.MessageID < IDPartitionSize*int(CONNECTION_REQ) {
			connectionReqAck <- ackMsg

		} else if ackMsg.MessageID < IDPartitionSize*int(CAB_REQ_INFO) {
			cabReqInfoAck <- ackMsg
		} else if ackMsg.MessageID < IDPartitionSize*int(HALL_ASSIGNMENT_COMPLETE) {
			hallAssignmentCompleteAck <- ackMsg
		}
	}
}

// Transmits Hall assignments from outgoingHallAssignments channel, and handles the ack
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
			newAssignment.MessageID = GenerateMessageID(NEW_HALL_ASSIGNMENT)

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

					// rebroadcast the message, and add a new timeout
					HallAssignmentsTx <- msg
					time.AfterFunc(time.Millisecond*500, func() {
						timeoutChannel <- msg.MessageID
					})
					break
				}
			}

		case receivedAck = <-HallAssignmentsAck:

			// check if message is in map, if not do nothing
			if msg, ok := activeAssignments[receivedAck.NodeID]; ok {
				if msg.MessageID == receivedAck.MessageID {

					delete(activeAssignments, receivedAck.MessageID)
				}
			}
		}

	}
}

// broadcasts the global hall requests with an interval, enable or disable by sending a bool in transmitEnableCh
func GlobalHallRequestsTransmitter(transmitEnableCh <-chan bool, GlobalHallRequestTx chan<- messages.GlobalHallRequest, requestsForBroadcastCh <-chan messages.GlobalHallRequest) {
	enable := false
	var GHallRequests messages.GlobalHallRequest

	for {
		select {

		case <-time.After(MASTER_TRANSMIT_INTERVAL):
		case enable = <-transmitEnableCh:
		case GHallRequests = <-requestsForBroadcastCh:
			if enable {
				GlobalHallRequestTx <- GHallRequests
			}
		}
	}
}

// server that tracks the states of all elevators by listening to the elevstatesrx channel
// you can requests to know the states by sending a string on  commandCh
// commands are "getActiveElevStates", "getActiveNodeIDs", "getAllKnownNodes", "getTOLC"
// known nodes includes both nodes that are considered active (you have recent contact) and "dead" nodes - previous contact have been made
func ElevStatesServer(commandCh <-chan string,
	timeOfLastContactCh chan<- time.Time,
	elevStatesCh chan<- map[int]messages.ElevStates,
	activeNodeIDsCh chan<- []int,
	elevStatesRx <-chan messages.ElevStates) {

	lastSeen := make(map[int]time.Time)
	knownNodes := make(map[int]messages.ElevStates)
	timeOfLastContact := time.Time{}

	for {
		select {

		// if you get a msg on elevStatesRx:
		case elevState := <-elevStatesRx:
			id := elevState.NodeID
			// here, we must check if the id is ours. Placeholder for MyID is 0 for now.
			if id != 0 {

				// My new time of last contact
				timeOfLastContact = time.Now()

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
						activeIDs = append(activeIDs, id)
					}
				}

				activeNodeIDsCh <- activeIDs

			case "getTOLC":
				timeOfLastContactCh <- timeOfLastContact

			case "getAllKnownNodes":
				elevStatesCh <- knownNodes
			}
		}
	}
}

// Transmits light updates when it receives them on outgoing light updates channel
// Waits for an ack on all transmitted messages, retransmits if no ack was received
func LightUpdateTransmitter(hallLightUpdateTx chan<- messages.HallLightUpdate,
	outgoingLightUpdates chan messages.HallLightUpdate,
	hallLightUpdateAck <-chan messages.Ack,
	commandCh chan<- string,
	activeNodeIDsCh <-chan []int) {

	activeAssignments := map[int]messages.HallLightUpdate{}

	timeoutCh := make(chan int)

	var timedOutMsgID int
	var receivedAck messages.Ack
	var newAssignment messages.HallLightUpdate

	for {
		select {
		case newAssignment = <-outgoingLightUpdates:

			// set a new message id
			newAssignment.MessageID = GenerateMessageID(HALL_LIGHT_UPDATE)

			commandCh <- "getActiveNodeIDs"
			activeNodeIDs := <-activeNodeIDsCh

			// set/overwrite old assignments
			for _, id := range activeNodeIDs {
				print(id)
				activeAssignments[id] = newAssignment
			}

			// send out the new assignment
			hallLightUpdateTx <- newAssignment

			// check for whether message is not acknowledged within duration
			time.AfterFunc(time.Millisecond*500, func() {
				timeoutCh <- newAssignment.MessageID
			})

		case timedOutMsgID = <-timeoutCh:

			// check if message is still in active assigments
			for _, msg := range activeAssignments {
				if msg.MessageID == timedOutMsgID {

					// send the message again
					hallLightUpdateTx <- msg
					time.AfterFunc(time.Millisecond*500, func() {
						timeoutCh <- newAssignment.MessageID
					})
					break
				}
			}

		case receivedAck = <-hallLightUpdateAck:
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

// temporary test function
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
	id, _ := strconv.Atoi(os.Args[1])

	go NetworkDude(id)
	for {
		time.Sleep(time.Second * 1000)
	}
}
