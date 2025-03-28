package communication

import (
	"elev/config"
	"elev/elevator"
	"elev/network/messages"
	"elev/util"
	"fmt"
	"maps"
	"math/rand"
	"time"
)

type NetworkEvent int

const (
	ActiveNodeCountChange NetworkEvent = iota
	NodeHasLostConnection
)

type UpdateType int

const (
	ActiveElevStates UpdateType = iota
	AllElevStates
	HallAssignmentRemoved
)

type ElevStateUpdate struct {
	NodeElevStatesMap map[int]elevator.ElevatorStateReport // Map keyed by node id, holding ElevatorStateReport
	DataType          UpdateType
}

// Generates a message ID
func GenerateMessageID() uint64 {
	return uint64(rand.Int63n(int64(config.MSG_ID_PARTITION_SIZE)))
}

// Server that tracks the states of all elevators by listening to the elevStatesRx channel
// It also can track whether the node is connected or disconnected
// If the number of nodes change, it will generate an event
// If there are hall assignments that are complete, it will generate an event
// You can requests to know the states by sending a string on commandCh
// Commands are "getActiveElevStates", "getAllKnownNodes", "startConnectionTimeoutDetection"
// Known nodes includes both nodes that are considered active (you have recent contact) and "dead" nodes - previous contact have been made
func ConnectionMonitorServer(myID int,
	commandRx <-chan string,
	elevStateUpdateTx chan<- ElevStateUpdate,
	elevStatesRx <-chan messages.NodeElevState,
	networkEventTx chan<- NetworkEvent) {

	nodeIsConnected := false
	connectionTimeoutTimer := time.NewTimer(config.NODE_CONNECTION_TIMEOUT)
	connectionTimeoutTimer.Stop()
	peerTimeoutTicker := time.NewTicker(config.PEER_POLL_INTERVAL)
	peerTimeoutTicker.Stop()

	lastSeen := make(map[int]time.Time)
	knownNodes := make(map[int]elevator.ElevatorStateReport)

	// Map of the last active nodes, only mutate this in the peer timeout ticker case
	lastActiveNodes := make(map[int]elevator.ElevatorStateReport)

	for {
		select {

		case <-peerTimeoutTicker.C:

			activeNodes := findActiveNodes(knownNodes, lastSeen)

			// If the number of active nodes change, generate an event
			if hasActiveNodesChanged(activeNodes, lastActiveNodes) {

				select {
				case networkEventTx <- ActiveNodeCountChange:
				default:
					fmt.Printf("Error sending network event\n")
				}
			}
			lastActiveNodes = activeNodes

		case <-connectionTimeoutTimer.C:
			// Node has timed out
			peerTimeoutTicker.Stop()
			nodeIsConnected = false

			// Set all the last seen times to zero
			lastActiveNodes = make(map[int]elevator.ElevatorStateReport)
			for id := range lastSeen {
				lastSeen[id] = time.Time{}
			}
			networkEventTx <- NodeHasLostConnection

		case elevState := <-elevStatesRx:
			id := elevState.NodeID

			if nodeIsConnectedToNetwork(id, myID, nodeIsConnected) { // If we are connected, we should update reset the connection timer
				connectionTimeoutTimer.Reset(config.NODE_CONNECTION_TIMEOUT)
			}

			// If I have seen this node before, check if it has cleared any hall assignments!
			if nodeExistInMap(id, knownNodes) {
				if util.HallAssignmentIsRemoved(knownNodes[id].MyHallAssignments, elevState.ElevState.MyHallAssignments) ||
					knownNodes[id].HACounterVersion != elevState.ElevState.HACounterVersion {
					// Update the lastActiveNodes with the new state, and send it to the node
					newActiveNodes := makeDeepCopy(lastActiveNodes)
					newActiveNodes[id] = elevState.ElevState
					elevStateUpdateTx <- makeHallAssignmentRemovedMessage(newActiveNodes)
				}
			}
			// Finally, register the node as seen
			lastSeen[id] = time.Now()
			knownNodes[id] = elevState.ElevState

		case command := <-commandRx:

			switch command {
			case "getActiveElevStates":
				elevStateUpdateTx <- makeActiveElevStatesUpdateMessage(lastActiveNodes)

			case "getAllElevStates":
				elevStateUpdateTx <- makeAllElevStatesUpdateMessage(knownNodes)

			case "startConnectionTimeoutDetection":
				fmt.Println("Starting connection timeout detection")
				connectionTimeoutTimer.Reset(config.NODE_CONNECTION_TIMEOUT)
				peerTimeoutTicker.Reset(config.PEER_POLL_INTERVAL)
				nodeIsConnected = true

			case "stopConnectionTimeoutDetection":
				connectionTimeoutTimer.Stop()
				peerTimeoutTicker.Stop()
				nodeIsConnected = false

				// Set all the last seen times to zero
				lastActiveNodes = make(map[int]elevator.ElevatorStateReport)
				for id := range lastSeen {
					lastSeen[id] = time.Time{}
				}
			}
		}
	}
}

func makeHallAssignmentRemovedMessage(elevStates map[int]elevator.ElevatorStateReport) ElevStateUpdate {
	return ElevStateUpdate{NodeElevStatesMap: elevStates, DataType: HallAssignmentRemoved}
}
func makeActiveElevStatesUpdateMessage(elevStates map[int]elevator.ElevatorStateReport) ElevStateUpdate {
	return ElevStateUpdate{NodeElevStatesMap: elevStates, DataType: ActiveElevStates}
}
func makeAllElevStatesUpdateMessage(elevStates map[int]elevator.ElevatorStateReport) ElevStateUpdate {
	return ElevStateUpdate{NodeElevStatesMap: elevStates, DataType: AllElevStates}
}

func findActiveNodes(knownNodes map[int]elevator.ElevatorStateReport, lastSeen map[int]time.Time) map[int]elevator.ElevatorStateReport {
	activeNodes := make(map[int]elevator.ElevatorStateReport)
	for id, t := range lastSeen {
		if time.Since(t) < config.NODE_CONNECTION_TIMEOUT {
			activeNodes[id] = knownNodes[id]
		}
	}
	return activeNodes
}

func makeDeepCopy(elevStateMap map[int]elevator.ElevatorStateReport) map[int]elevator.ElevatorStateReport {
	newMap := make(map[int]elevator.ElevatorStateReport)
	maps.Copy(newMap, elevStateMap)
	return newMap
}

func nodeIsConnectedToNetwork(msgID, myID int, nodeIsConnected bool) bool {
	return nodeIsConnected && msgID != myID
}

func hasActiveNodesChanged(activeNodes, lastActiveNodes map[int]elevator.ElevatorStateReport) bool {
	return len(activeNodes) != len(lastActiveNodes) && (len(activeNodes) > 1)
}

func nodeExistInMap(id int, knownNodes map[int]elevator.ElevatorStateReport) bool {
	_, ok := knownNodes[id]
	return ok
}
