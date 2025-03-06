package messageHandler

import (
	"elev/Network/messages"
	"elev/util/config"
	"errors"
	"math/rand"
	"time"
)

type MessageIDType uint64

const (
	NEW_HALL_ASSIGNMENT      MessageIDType = 0
	HALL_LIGHT_UPDATE        MessageIDType = 1
	CONNECTION_REQ           MessageIDType = 2
	HALL_ASSIGNMENT_COMPLETE MessageIDType = 3
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

// Listens to incoming acknowledgment messages from UDP, distributes them to their corresponding channels
func IncomingAckDistributor(ackRx <-chan messages.Ack,
	hallAssignmentsAck chan<- messages.Ack,
	lightUpdateAck chan<- messages.Ack,
	connectionReqAck chan<- messages.Ack,
	hallAssignmentCompleteAck chan<- messages.Ack) {

	for ackMsg := range ackRx {

		if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(uint64(NEW_HALL_ASSIGNMENT)+1) {
			hallAssignmentsAck <- ackMsg

		} else if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(uint64(HALL_LIGHT_UPDATE)+1) {
			lightUpdateAck <- ackMsg

		} else if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(uint64(CONNECTION_REQ)+1) {
			connectionReqAck <- ackMsg

		} else if ackMsg.MessageID < config.MSG_ID_PARTITION_SIZE*(uint64(HALL_ASSIGNMENT_COMPLETE)+1) {
			hallAssignmentCompleteAck <- ackMsg
		}
	}
}

// server that tracks the states of all elevators by listening to the elevStatesRx channel
// you can requests to know the states by sending a string on  commandCh
// commands are "getActiveElevStates", "getActiveNodeIDs", "getAllKnownNodes", "getTOLC", "startConnectionTimeoutDetection"
// known nodes includes both nodes that are considered active (you have recent contact) and "dead" nodes - previous contact have been made
func NodeElevStateServer(myID int, commandRx <-chan string,
	timeOfLastContactTx chan<- time.Time,
	activeElevStatesTx chan<- map[int]messages.NodeElevState,
	activeNodeIDsTx chan<- []int,
	elevStatesRx <-chan messages.NodeElevState,
	allElevStatesTx chan<- map[int]messages.NodeElevState,
	connectionTimeoutEventTx chan<- bool,
) {
	// go routine is structured around its data. It is responsible for collecting it and remembering  it

	enableTOLCUpdate := false
	timeoutTimer := time.NewTimer(config.NODE_CONNECTION_TIMEOUT)
	timeoutTimer.Stop()

	lastSeen := make(map[int]time.Time)
	knownNodes := make(map[int]messages.NodeElevState)
	timeOfLastContact := time.Time{}

	for {
		select {

		case <-timeoutTimer.C:
			enableTOLCUpdate = false
			connectionTimeoutEventTx <- true

		case elevState := <-elevStatesRx:
			id := elevState.NodeID
			if id != myID { // Check if we received our own message

				if enableTOLCUpdate {
					timeOfLastContact = time.Now()
					timeoutTimer.Reset(config.NODE_CONNECTION_TIMEOUT)
				}

				knownNodes[id] = elevState
				lastSeen[id] = time.Now()
			}

		case command := <-commandRx:

			switch command {
			case "getActiveElevStates":
				activeNodes := make(map[int]messages.NodeElevState)
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

			case "startConnectionTimeoutDetection":
				timeoutTimer.Reset(config.NODE_CONNECTION_TIMEOUT)
				enableTOLCUpdate = true

			}
		}
	}
}
