package comm

import (
	"elev/Network/network/messages"
	"elev/util/config"
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

// Listens to incoming acknowledgment messages from UDP, distributes them to their corresponding channels
func IncomingAckDistributor(ackRx <-chan messages.Ack,
	hallAssignmentsAck chan<- messages.Ack,
	lightUpdateAck chan<- messages.Ack,
	connectionReqAck chan<- messages.Ack,
	cabReqInfoAck chan<- messages.Ack,
	hallAssignmentCompleteAck chan<- messages.Ack) {

	for ackMsg := range ackRx {

		if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(uint64(NEW_HALL_ASSIGNMENT)+1) {
			hallAssignmentsAck <- ackMsg

		} else if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(uint64(HALL_LIGHT_UPDATE)+1) {
			lightUpdateAck <- ackMsg

		} else if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(uint64(CONNECTION_REQ)+1) {
			connectionReqAck <- ackMsg

		} else if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(uint64(CAB_REQ_INFO)+1) {
			cabReqInfoAck <- ackMsg

		} else if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(uint64(HALL_ASSIGNMENT_COMPLETE)+1) {
			hallAssignmentCompleteAck <- ackMsg
		}
	}
}

// server that tracks the states of all elevators by listening to the elevStatesRx channel
// you can requests to know the states by sending a string on  commandCh
// commands are "getActiveElevStates", "getActiveNodeIDs", "getAllKnownNodes", "getTOLC"
// known nodes includes both nodes that are considered active (you have recent contact) and "dead" nodes - previous contact have been made
func ElevStatesListener(commandRx <-chan string,
	timeOfLastContactTx chan<- time.Time,
	activeElevStatesTx chan<- map[int]messages.ElevStates,
	activeNodeIDsTx chan<- []int,
	elevStatesRx <-chan messages.ElevStates,
	allElevStatesTx chan<- map[int]messages.ElevStates) {
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

		case command := <-commandRx:

			switch command {
			case "getActiveElevStates":

				activeNodes := make(map[int]messages.ElevStates)
				for id, t := range lastSeen {
					if time.Since(t) < config.CONNECTION_TIMEOUT {
						activeNodes[id] = knownNodes[id]
					}
				}
				activeElevStatesTx <- activeNodes

			case "getActiveNodeIDs":

				activeIDs := make([]int, 0)
				for id, t := range lastSeen {
					if time.Since(t) < config.CONNECTION_TIMEOUT {
						activeIDs = append(activeIDs, id)
					}
				}

				activeNodeIDsTx <- activeIDs

			case "getTOLC":
				timeOfLastContactTx <- timeOfLastContact

			case "getAllElevStates":
				allElevStatesTx <- knownNodes
			}
		}
	}
}
