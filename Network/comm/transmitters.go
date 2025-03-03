package comm

import (
	"elev/Network/network/messages"
	"elev/util/config"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type MessageIDType uint64

const (
	NEW_HALL_ASSIGNMENT      MessageIDType = 0
	HALL_LIGHT_UPDATE        MessageIDType = 1
	CONNECTION_REQ           MessageIDType = 2
	CAB_REQ_INFO             MessageIDType = 3
	HALL_ASSIGNMENT_COMPLETE MessageIDType = 4
)

// generates a message ID that corresponsds to the message type
func GenerateMessageID(partition MessageIDType) (uint64, error) {
	offset := uint64(partition)

	if offset > uint64(HALL_ASSIGNMENT_COMPLETE) {
		return 0, errors.New("invalid messageIDType")
	}

	i := uint64(rand.Int63n(int64(config.MSG_ID_PARTITION_SIZE)))
	i += uint64((config.MSG_ID_PARTITION_SIZE) * offset)

	return i, nil
}

// Transmits Hall assignments from outgoingHallAssignments channel to their designated elevators and handles ack
func HallAssignmentsTransmitter(HallAssignmentsTx chan<- messages.NewHallAssignments,
	OutgoingNewHallAssignments <-chan messages.NewHallAssignments,
	HallAssignmentsAck <-chan messages.Ack) {

	activeAssignments := map[int]messages.NewHallAssignments{}

	timeoutChannel := make(chan uint64, 2)

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

// Transmits HallButton Lightstates from outgoingLightUpdates channel to their designated elevators and handles ack
func LightUpdateTransmitter(hallLightUpdateTx chan<- messages.HallLightUpdate,
	outgoingLightUpdates chan messages.HallLightUpdate,
	hallLightUpdateAck <-chan messages.Ack) {

	activeAssignments := map[int]messages.HallLightUpdate{}
	timeoutCh := make(chan uint64)

	for {
		select {
		case newLightUpdate := <-outgoingLightUpdates:

			new_msg_id, err := GenerateMessageID(HALL_LIGHT_UPDATE)
			if err != nil {
				fmt.Println("Fatal error, invalid message type used to generate message id in hall light update")
			}

			newLightUpdate.MessageID = new_msg_id

			// make the actual message shorter by removing redundant information

			for _, id := range newLightUpdate.ActiveElevatorIDs {
				activeAssignments[id] = newLightUpdate
			}

			newLightUpdate.ActiveElevatorIDs = []int{}

			hallLightUpdateTx <- newLightUpdate

			time.AfterFunc(time.Millisecond*500, func() {
				timeoutCh <- newLightUpdate.MessageID
			})

		case timedOutMsgID := <-timeoutCh:

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

		case receivedAck := <-hallLightUpdateAck:

			if msg, ok := activeAssignments[receivedAck.NodeID]; ok {
				if msg.MessageID == receivedAck.MessageID {

					delete(activeAssignments, receivedAck.NodeID)
				}
			}
		}
	}
}
