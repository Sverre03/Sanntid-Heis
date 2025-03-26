package messagehandler

import (
	"elev/Network/messages"
	"elev/config"
	"elev/elevator"
	"fmt"
	"maps"
	"math/rand"
	"time"
)

type MessageIDType uint64

const (
	NEW_HALL_ASSIGNMENT MessageIDType = 0
	HALL_LIGHT_UPDATE   MessageIDType = 1
	CONNECTION_REQ      MessageIDType = 2
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
	NodeElevStatesMap map[int]elevator.ElevatorState // map keyed by node id, holding ElevatorStates
	DataType          UpdateType
}

// generates a message ID that corresponsds to the message type
func GenerateMessageID(partition MessageIDType) uint64 {
	return uint64(rand.Int63n(int64(config.MSG_ID_PARTITION_SIZE)))
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

	nodeIsConnected := false
	connectionTimeoutTimer := time.NewTimer(config.NODE_CONNECTION_TIMEOUT)
	connectionTimeoutTimer.Stop()
	peerTimeoutTicker := time.NewTicker(config.PEER_POLL_INTERVAL)
	peerTimeoutTicker.Stop()

	lastSeen := make(map[int]time.Time)
	knownNodes := make(map[int]elevator.ElevatorState)

	// map of the last active nodes, only mutate this in the peer timeout ticker case
	lastActiveNodes := make(map[int]elevator.ElevatorState)

	for {
		select {

		case <-peerTimeoutTicker.C:

			activeNodes := findActiveNodes(knownNodes, lastSeen)

			// if the number of active nodes change, generate an event
			if hasActiveNodesChanged(activeNodes, lastActiveNodes) && len(activeNodes) > 1 {
				fmt.Printf("Active nodes changed from %d to %d\n", len(lastActiveNodes), len(activeNodes))
				select {
				case networkEventTx <- ActiveNodeCountChange:

				default:
					fmt.Printf("Error sending network event\n")
				}
			}
			lastActiveNodes = activeNodes

		case <-connectionTimeoutTimer.C:
			// we have timed out
			peerTimeoutTicker.Stop()
			nodeIsConnected = false

			// set all the last seen times to zero
			lastActiveNodes = make(map[int]elevator.ElevatorState)
			for id := range lastSeen {
				lastSeen[id] = time.Time{}
			}
			networkEventTx <- NodeHasLostConnection

		case elevState := <-elevStatesRx:
			id := elevState.NodeID

			if nodeIsConnectedToNetwork(id, myID, nodeIsConnected) { // if we are connected, we should update reset the connection timer
				connectionTimeoutTimer.Reset(config.NODE_CONNECTION_TIMEOUT)
			}

			// if I have seen this node before, check if it has cleared any hall assignments!
			if nodeExistInMap(id, knownNodes) && hallAssignmentIsRemoved(knownNodes[id].MyHallAssignments, elevState.ElevState.MyHallAssignments) {
				// update the lastActiveNodes with the new state, and send it to the node
				newActiveNodes := makeDeepCopy(lastActiveNodes)
				newActiveNodes[id] = elevState.ElevState
				elevStateUpdateTx <- makeHallAssignmentRemovedMessage(newActiveNodes)

				// fmt.Printf(("Hall assignment removed by node %d\n"), id)
			}
			// finally, register the node as seen
			lastSeen[id] = time.Now()
			knownNodes[id] = elevState.ElevState

		case command := <-commandRx:

			switch command {
			case "getActiveElevStates":
				elevStateUpdateTx <- makeActiveElevStatesUpdateMessage(lastActiveNodes)

			case "getAllElevStates":
				elevStateUpdateTx <- makeAllElevStatesUpdateMessage(knownNodes)

			case "startConnectionTimeoutDetection":
				connectionTimeoutTimer.Reset(config.NODE_CONNECTION_TIMEOUT)
				peerTimeoutTicker.Reset(config.PEER_POLL_INTERVAL)
				nodeIsConnected = true

			case "stopConnectionTimeoutDetection":
				connectionTimeoutTimer.Stop()
				peerTimeoutTicker.Stop()
				nodeIsConnected = false

				// set all the last seen times to zero
				lastActiveNodes = make(map[int]elevator.ElevatorState)
				for id := range lastSeen {
					lastSeen[id] = time.Time{}
				}
			}
		}
	}
}

func makeHallAssignmentRemovedMessage(elevStates map[int]elevator.ElevatorState) ElevStateUpdate {
	return ElevStateUpdate{NodeElevStatesMap: elevStates, DataType: HallAssignmentRemoved}
}

func makeActiveElevStatesUpdateMessage(elevStates map[int]elevator.ElevatorState) ElevStateUpdate {
	return ElevStateUpdate{NodeElevStatesMap: elevStates, DataType: ActiveElevStates}
}
func makeAllElevStatesUpdateMessage(elevStates map[int]elevator.ElevatorState) ElevStateUpdate {
	return ElevStateUpdate{NodeElevStatesMap: elevStates, DataType: AllElevStates}
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

func hallAssignmentIsRemoved(oldGlobalHallRequests [config.NUM_FLOORS][2]bool,
	newGlobalHallReq [config.NUM_FLOORS][2]bool) bool {
	for floor := range config.NUM_FLOORS {
		for button := range 2 {
			// Check if change is from (true -> false), assignment complete
			if oldGlobalHallRequests[floor][button] && !newGlobalHallReq[floor][button] {
				// fmt.Printf("Hall assignment removed at floor %d, button %d\n", floor, button)
				return true
			}
		}
	}
	return false
}

func makeDeepCopy(elevStateMap map[int]elevator.ElevatorState) map[int]elevator.ElevatorState {
	newMap := make(map[int]elevator.ElevatorState)
	maps.Copy(newMap, elevStateMap)
	return newMap
}

func nodeIsConnectedToNetwork(myID int, msgID int, nodeIsConnected bool) bool {
	return nodeIsConnected && myID != msgID
}

func hasActiveNodesChanged(activeNodes map[int]elevator.ElevatorState, lastActiveNodes map[int]elevator.ElevatorState) bool {
	return len(activeNodes) != len(lastActiveNodes)
}

func nodeExistInMap(id int, knownNodes map[int]elevator.ElevatorState) bool {
	_, ok := knownNodes[id]
	return ok
}
