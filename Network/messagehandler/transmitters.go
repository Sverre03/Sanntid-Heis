package messagehandler

import (
	"elev/Network/messages"
	"elev/config"
	"time"
)

// Transmits Hall assignments from OutgoingHallAssignments channel to their designated elevators and handles ack - i.e resends if the message didnt arrive
func HallAssignmentsTransmitter(HallAssignmentsTx chan<- messages.NewHallAssignments,
	OutgoingNewHallAssignments <-chan messages.NewHallAssignments,
	HallAssignmentsAck <-chan messages.Ack,
	HallAssignerEnableCh <-chan bool) {

	activeAssignments := map[int]messages.NewHallAssignments{}
	timeoutChannel := make(chan uint64, 2)
	enable := false

	for {
		select {
		case enable = <-HallAssignerEnableCh:
			if !enable {
				clearActiveAssignments(activeAssignments)
			}
		case newAssignment := <-OutgoingNewHallAssignments:
			if !enable {
				continue
			}
			newAssignment.MessageID = GenerateMessageID(NEW_HALL_ASSIGNMENT)
			activeAssignments[newAssignment.NodeID] = newAssignment

			HallAssignmentsTx <- newAssignment

			// Resend the message if no ack is received within the timeout
			time.AfterFunc(config.HALL_ASSIGNMENT_ACK_TIMEOUT, func() {
				timeoutChannel <- newAssignment.MessageID
			})

		case timedOutMsgID := <-timeoutChannel:
			for _, msg := range activeAssignments {
				if msg.MessageID == timedOutMsgID {
					HallAssignmentsTx <- msg
					time.AfterFunc(config.HALL_ASSIGNMENT_ACK_TIMEOUT, func() {
						timeoutChannel <- msg.MessageID
					})
					break
				}
			}

		case receivedAck := <-HallAssignmentsAck:
			if msg, ok := activeAssignments[receivedAck.NodeID]; ok {
				if msg.MessageID == receivedAck.MessageID {
					delete(activeAssignments, receivedAck.NodeID)
				}
			}
		}

	}
}

// broadcasts the global hall requests with an interval, enable or disable by sending a bool in transmitEnableCh
func GlobalHallRequestsTransmitter(transmitEnableCh <-chan bool,
	GlobalHallRequestTx chan<- messages.GlobalHallRequest,
	requestsForBroadcastCh <-chan messages.GlobalHallRequest) {

	enable := false
	var currentRequests messages.GlobalHallRequest

	for {
		select {
		case enable = <-transmitEnableCh:
		case currentRequests = <-requestsForBroadcastCh:
		case <-time.After(config.MASTER_TRANSMIT_INTERVAL):
			if enable {
				GlobalHallRequestTx <- currentRequests
			}
		}
	}
}

func clearActiveAssignments(activeAssignments map[int]messages.NewHallAssignments) {
	for key := range activeAssignments {
		delete(activeAssignments, key)
	}
}
