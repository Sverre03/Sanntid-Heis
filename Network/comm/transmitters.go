package comm

import (
	"elev/Network/network/messages"
	"elev/util/config"
	"fmt"
	"time"
)

// Transmits Hall assignments from outgoingHallAssignments channel to their designated elevators and handles ack
func HallAssignmentsTransmitter(HallAssignmentsTx chan<- messages.NewHallAssignments,
	OutgoingNewHallAssignments <-chan messages.NewHallAssignments,
	HallAssignmentsAck <-chan messages.Ack) {

	activeAssignments := map[int]messages.NewHallAssignments{}

	timeoutChannel := make(chan uint64, 2)

	for {
		select {
		case newAssignment := <-OutgoingNewHallAssignments:
			//fmt.Printf("got new hall assignment with id %d\n", newAssignment.NodeID)
			new_msg_id, err := GenerateMessageID(NEW_HALL_ASSIGNMENT)
			if err != nil {
				fmt.Println("Fatal error, invalid message id type used to generate a message id in HallAssignmentTransmitter")
			}

			newAssignment.MessageID = new_msg_id

			// fmt.Printf("got new hall assignment with id %d and a message id %d\n", newAssignment.NodeID, newAssignment.MessageID)
			activeAssignments[newAssignment.NodeID] = newAssignment
			//fmt.Printf("active assignments: %v\n", activeAssignments[newAssignment.NodeID])
			HallAssignmentsTx <- newAssignment

			// check for whether message is not acknowledged within duration
			time.AfterFunc(500*time.Millisecond, func() {
				timeoutChannel <- newAssignment.MessageID
			})

		case timedOutMsgID := <-timeoutChannel:

			// fmt.Printf("Checking messageID for resend: %d \n", timedOutMsgID)
			for _, msg := range activeAssignments {
				if msg.MessageID == timedOutMsgID {

					// fmt.Printf("resending message id %d \n", timedOutMsgID)
					HallAssignmentsTx <- msg
					time.AfterFunc(500*time.Millisecond, func() {
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

			time.AfterFunc(500*time.Millisecond, func() {
				timeoutCh <- newLightUpdate.MessageID
			})

		case timedOutMsgID := <-timeoutCh:

			for _, msg := range activeAssignments {
				if msg.MessageID == timedOutMsgID {

					// send the message again
					hallLightUpdateTx <- msg
					time.AfterFunc(500*time.Millisecond, func() {
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

func HallAssignmentCompleteTransmitter(HallAssignmentCompleteTx chan<- messages.HallAssignmentComplete, 
	hallAssignmentCompleteRx <-chan messages.HallAssignmentComplete, 
	hallAssignmentCompleteAckRx <-chan messages.Ack) {
	
	timeoutChannel := make(chan uint64, 2)
	completedActiveAssignments := make(map[uint64]messages.HallAssignmentComplete) //mapping message id to hall assignment complete message
	
	for {
		select {
			case newComplete := <-hallAssignmentCompleteRx:
				new_msg_id, err := GenerateMessageID(HALL_ASSIGNMENT_COMPLETE)
				if err != nil {
					fmt.Println("Fatal error, invalid message type used to generate message id in hall assignment complete")
				}
				newComplete.MessageID = new_msg_id
				completedActiveAssignments[new_msg_id] = newComplete
				fmt.Printf("Hall Assignment %v is completed\n", newComplete)
				HallAssignmentCompleteTx <- newComplete

				time.AfterFunc(500*time.Millisecond, func() {
					timeoutChannel <- newComplete.MessageID
				})
			case receivedAck := <-hallAssignmentCompleteAckRx:
				if msg, ok := completedActiveAssignments[receivedAck.MessageID]; ok {
					if msg.MessageID == receivedAck.MessageID {
						delete(completedActiveAssignments, receivedAck.MessageID)
					}
				}
			case timedOutMsgID := <-timeoutChannel:
				
				for _, msg := range completedActiveAssignments {
					if msg.MessageID == timedOutMsgID {
	
						// fmt.Printf("resending message id %d \n", timedOutMsgID)
						HallAssignmentCompleteTx <- msg
						time.AfterFunc(500*time.Millisecond, func() {
							timeoutChannel <- msg.MessageID
						})
						break
					}
				}
	
			
		}
	}
}
