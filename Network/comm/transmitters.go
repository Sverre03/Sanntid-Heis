package comm

import (
	"elev/Network/network/messages"
	"elev/util/config"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type MessageIDType int

const (
	NEW_HALL_ASSIGNMENT      MessageIDType = 0
	HALL_LIGHT_UPDATE        MessageIDType = 1
	CONNECTION_REQ           MessageIDType = 2
	CAB_REQ_INFO             MessageIDType = 3
	HALL_ASSIGNMENT_COMPLETE MessageIDType = 4
)

// generates a message ID that corresponsds to the message type
func GenerateMessageID(partition MessageIDType) (int, error) {
	offset := int(partition)

	if offset > int(HALL_ASSIGNMENT_COMPLETE) || offset < 0 {
		return 0, errors.New("invalid messageIDType")
	}

	i := rand.Intn(config.MSG_ID_PARTITION_SIZE)
	i += (config.MSG_ID_PARTITION_SIZE) * offset

	return i, nil
}

// Listens to incoming acknowledgment messages from UDP, distributes them to their corresponding channels
func IncomingAckDistributor(ackRx <-chan messages.Ack,
	hallAssignmentsAck chan<- messages.Ack,
	lightUpdateAck chan<- messages.Ack,
	connectionReqAck chan<- messages.Ack,
	cabReqInfoAck chan<- messages.Ack,
	hallAssignmentCompleteAck chan<- messages.Ack) {

	for ackMsg := range ackRx {

		if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(int(NEW_HALL_ASSIGNMENT)+1) {
			hallAssignmentsAck <- ackMsg

		} else if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(int(HALL_LIGHT_UPDATE)+1) {
			lightUpdateAck <- ackMsg

		} else if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(int(CONNECTION_REQ)+1) {
			connectionReqAck <- ackMsg

		} else if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(int(CAB_REQ_INFO)+1) {
			cabReqInfoAck <- ackMsg

		} else if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(int(HALL_ASSIGNMENT_COMPLETE)+1) {
			hallAssignmentCompleteAck <- ackMsg
		}
	}
}

// Transmits Hall assignments from outgoingHallAssignments channel to their designated elevators and handles ack
func HallAssignmentsTransmitter(HallAssignmentsTx chan<- messages.NewHallAssignments,
	OutgoingNewHallAssignments <-chan messages.NewHallAssignments,
	HallAssignmentsAck <-chan messages.Ack) {

	activeAssignments := map[int]messages.NewHallAssignments{}

	timeoutChannel := make(chan int, 2)

	for {
		select {
		case newAssignment := <-OutgoingNewHallAssignments:

			new_msg_id, err := GenerateMessageID(NEW_HALL_ASSIGNMENT)
			if err != nil {
				fmt.Println("Fatal error, invalid message id type used to generate a message id in HallAssignmentTransmitter")
			}

			newAssignment.MessageID = new_msg_id

			// fmt.Printf("got new hall assignment with id %d and a message id %d\n", newAssignment.NodeID, newAssignment.MessageID)
			activeAssignments[newAssignment.NodeID] = newAssignment

			HallAssignmentsTx <- newAssignment

			// check for whether message is not acknowledged within duration
			time.AfterFunc(time.Millisecond*500, func() {
				timeoutChannel <- newAssignment.MessageID
			})

		case timedOutMsgID := <-timeoutChannel:

			// fmt.Printf("Checking messageID for resend: %d \n", timedOutMsgID)
			for _, msg := range activeAssignments {
				if msg.MessageID == timedOutMsgID {

					// fmt.Printf("resending message id %d \n", timedOutMsgID)
					HallAssignmentsTx <- msg
					time.AfterFunc(time.Millisecond*500, func() {
						timeoutChannel <- msg.MessageID
					})
					break
				}
			}

		case receivedAck := <-HallAssignmentsAck:
			if msg, ok := activeAssignments[receivedAck.NodeID]; ok {
				if msg.MessageID == receivedAck.MessageID {
					// fmt.Printf("Deleting assignment with node id %d and message id %d \n", receivedAck.NodeID, receivedAck.MessageID)
					delete(activeAssignments, receivedAck.NodeID)
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

		case GHallRequests = <-requestsForBroadcastCh:
		case enable = <-transmitEnableCh:
		case <-time.After(config.MASTER_TRANSMIT_INTERVAL):
			if enable {
				GlobalHallRequestTx <- GHallRequests
			}
		}
	}
}

// server that tracks the states of all elevators by listening to the elevStatesRx channel
// you can requests to know the states by sending a string on  commandCh
// commands are "getActiveElevStates", "getActiveNodeIDs", "getAllKnownNodes", "getTOLC"
// known nodes includes both nodes that are considered active (you have recent contact) and "dead" nodes - previous contact have been made
func ElevStatesListener(commandCh <-chan string,
	timeOfLastContactCh chan<- time.Time,
	elevStatesCh chan<- map[int]messages.ElevStates,
	activeNodeIDsCh chan<- []int,
	elevStatesRx <-chan messages.ElevStates) {
	// go routine is structured around its data. It is responsible for collecting it and remembering  it

	lastSeen := make(map[int]time.Time)
	knownNodes := make(map[int]messages.ElevStates)
	timeOfLastContact := time.Time{}

	for {
		select {

		case elevState := <-elevStatesRx:
			id := elevState.NodeID

			// here, we must check if the id is ours. Placeholder for MyID is 0 for now.
			if id != 0 {
				timeOfLastContact = time.Now()

				knownNodes[id] = elevState
				lastSeen[id] = time.Now()
			}

		case command := <-commandCh:

			switch command {
			case "getActiveElevStates":

				activeNodes := make(map[int]messages.ElevStates)
				for id, t := range lastSeen {
					if time.Since(t) < config.CONNECTION_TIMEOUT {
						activeNodes[id] = knownNodes[id]
					}
				}
				elevStatesCh <- activeNodes

			case "getActiveNodeIDs":

				activeIDs := make([]int, 0)
				for id, t := range lastSeen {
					if time.Since(t) < config.CONNECTION_TIMEOUT {
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

// Transmits HallButton Lightstates from outgoingLightUpdates channel to their designated elevators and handles ack
func LightUpdateTransmitter(hallLightUpdateTx chan<- messages.HallLightUpdate,
	outgoingLightUpdates chan messages.HallLightUpdate,
	hallLightUpdateAck <-chan messages.Ack,
	commandCh chan<- string,
	activeNodeIDsCh <-chan []int) {

	activeAssignments := map[int]messages.HallLightUpdate{}

	timeoutCh := make(chan int)

	var timedOutMsgID int
	var receivedAck messages.Ack
	var newLightUpdate messages.HallLightUpdate

	for {
		select {
		case newLightUpdate = <-outgoingLightUpdates:

			new_msg_id, err := GenerateMessageID(HALL_LIGHT_UPDATE)
			if err != nil {
				fmt.Println("Fatal error, invalid message type used to generate message id")
			}

			newLightUpdate.MessageID = new_msg_id

			for _, id := range newLightUpdate.ActiveElevatorIDs {
				activeAssignments[id] = newLightUpdate
			}

			hallLightUpdateTx <- newLightUpdate

			time.AfterFunc(time.Millisecond*500, func() {
				timeoutCh <- newLightUpdate.MessageID
			})

		case timedOutMsgID = <-timeoutCh:

			for _, msg := range activeAssignments {
				if msg.MessageID == timedOutMsgID {

					// send the message again
					hallLightUpdateTx <- msg
					time.AfterFunc(time.Millisecond*500, func() {
						timeoutCh <- msg.MessageID
					})
					break
				}
			}

		case receivedAck = <-hallLightUpdateAck:

			if msg, ok := activeAssignments[receivedAck.NodeID]; ok {
				if msg.MessageID == receivedAck.MessageID {

					delete(activeAssignments, receivedAck.MessageID)
				}
			}
		}
	}
}
