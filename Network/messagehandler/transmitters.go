package messagehandler

import (
	"elev/Network/messages"
	"elev/config"
	"time"
)

// Transmits Hall assignments from outgoingHallAssignments channel to their designated elevators and handles ack - i.e resends if the message didnt arrive
func HallAssignmentsTransmitter(HallAssignmentsTx chan<- messages.NewHallAssignments,
	OutgoingNewHallAssignments <-chan messages.NewHallAssignments,
	HallAssignmentsAck <-chan messages.Ack,
	HallAssignerEnableCH <-chan bool) {

	activeAssignments := map[int]messages.NewHallAssignments{}
	timeoutChannel := make(chan uint64, 2)
	enable := false

	for {
	Select:
		select {
		case enable = <-HallAssignerEnableCH:
			if !enable {
				for k := range activeAssignments {
					delete(activeAssignments, k)
				}
			}
		case newAssignment := <-OutgoingNewHallAssignments:
			if !enable {
				break Select
			}
			//fmt.Printf("got new hall assignment with id %d\n", newAssignment.NodeID)
			new_msg_id := GenerateMessageID(NEW_HALL_ASSIGNMENT)

			newAssignment.MessageID = new_msg_id

			// fmt.Printf("got new hall assignment with id %d and a message id %d\n", newAssignment.NodeID, newAssignment.MessageID)
			activeAssignments[newAssignment.NodeID] = newAssignment
			//fmt.Printf("active assignments: %v\n", activeAssignments[newAssignment.NodeID])
			HallAssignmentsTx <- newAssignment

			// check for whether message is not acknowledged within duration
			time.AfterFunc(config.HALL_ASSIGNMENT_ACK_TIMEOUT, func() {
				timeoutChannel <- newAssignment.MessageID
			})

		case timedOutMsgID := <-timeoutChannel:

			// fmt.Printf("Checking messageID for resend: %d \n", timedOutMsgID)
			for _, msg := range activeAssignments {
				if msg.MessageID == timedOutMsgID {

					// fmt.Printf("resending message id %d \n", timedOutMsgID)
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
		case enable = <-transmitEnableCh:
		case GHallRequests = <-requestsForBroadcastCh:
		case <-time.After(config.MASTER_TRANSMIT_INTERVAL):
			if enable {
				GlobalHallRequestTx <- GHallRequests
			}
		}
	}
}
