package messagehandler

import (
	"elev/Network/messages"
	"elev/elevator"
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
	HALL_ASSIGNMENT_COMPLETE MessageIDType = 3
)

type NetworkEvent int

const (
	NodeConnectDisconnect NetworkEvent = iota
	NodeHasLostConnection
)

type ElevStateUpdate struct {
	NodeElevStatesMap map[int]elevator.ElevatorState // map keyed by node id, holding ElevatorStates
	OnlyActiveNodes   bool
}

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
// commands are "getActiveElevStates", "getAllKnownNodes", "startConnectionTimeoutDetection"
// known nodes includes both nodes that are considered active (you have recent contact) and "dead" nodes - previous contact have been made
func NodeElevStateServer(myID int,
	commandRx <-chan string,
	elevStateUpdateTx chan<- ElevStateUpdate,
	elevStatesRx <-chan messages.NodeElevState,
	networkEventTx chan<- NetworkEvent,
) {
	// go routine is structured around its data. It is responsible for collecting it and remembering  it

	nodeIsConnected := false
	connectionTimeoutTimer := time.NewTicker(config.NODE_CONNECTION_TIMEOUT)
	connectionTimeoutTimer.Stop()
	peerTimeoutTicker := time.NewTimer(config.PEER_POLL_INTERVAL)

	peerTimeoutTicker.Stop()
	lastSeen := make(map[int]time.Time)
	knownNodes := make(map[int]elevator.ElevatorState)

	lastActiveNodes := make(map[int]elevator.ElevatorState)
	for {
		select {
		case <-peerTimeoutTicker.C:
			activeNodes := findActiveNodes(knownNodes, lastSeen)

			// if the number of active nodes change, generate an event
			if len(lastActiveNodes) != len(activeNodes) {

				networkEventTx <- NodeConnectDisconnect
			}
			lastActiveNodes = activeNodes

		case <-connectionTimeoutTimer.C:
			// we have timed out
			peerTimeoutTicker.Stop()
			nodeIsConnected = false

			// we have lost connection. Empty the lastActiveNodes map, just in case some strays are still left there
			lastActiveNodes = make(map[int]elevator.ElevatorState)
			networkEventTx <- NodeHasLostConnection

		case elevState := <-elevStatesRx:
			id := elevState.NodeID
			if id != myID { // Check if we received our own message
				if nodeIsConnected {
					connectionTimeoutTimer.Reset(config.NODE_CONNECTION_TIMEOUT)
				}

				knownNodes[id] = elevState.ElevState
				lastSeen[id] = time.Now()
			}

		case command := <-commandRx:

			switch command {
			case "getActiveElevStates":
				activeNodes := findActiveNodes(knownNodes, lastSeen)

				elevStateUpdateTx <- makeActiveElevStatesUpdateMessage(activeNodes)

			case "getAllElevStates":
				elevStateUpdateTx <- makeAllElevStatesUpdateMessage(knownNodes)

			case "startConnectionTimeoutDetection":
				connectionTimeoutTimer.Reset(config.NODE_CONNECTION_TIMEOUT)
				peerTimeoutTicker.Reset(config.PEER_POLL_INTERVAL)
				nodeIsConnected = true
				fmt.Printf("Node %d connection detection routine started\n", myID)
			}
		}
	}
}

func makeActiveElevStatesUpdateMessage(elevStates map[int]elevator.ElevatorState) ElevStateUpdate {
	return ElevStateUpdate{NodeElevStatesMap: elevStates, OnlyActiveNodes: true}
}
func makeAllElevStatesUpdateMessage(elevStates map[int]elevator.ElevatorState) ElevStateUpdate {
	return ElevStateUpdate{NodeElevStatesMap: elevStates, OnlyActiveNodes: false}
}

func findActiveNodes(knownNodes map[int]elevator.ElevatorState, lastSeen map[int]time.Time) map[int]elevator.ElevatorState {
	activeNodes := make(map[int]elevator.ElevatorState)
	for id, t := range lastSeen {
		if time.Since(t) < config.NODE_CONNECTION_TIMEOUT {
			activeNodes[id] = knownNodes[id]
		}
	}
	return activeNodes
}
