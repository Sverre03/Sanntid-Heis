package node

import (
	"elev/Network/messagehandler"
	"elev/Network/messages"
	"elev/config"
	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"elev/singleelevator"
	"elev/util"
	"fmt"
	"time"
)

func MasterProgram(node *NodeData) nodestate {
	fmt.Printf("Initiating master: Global requests: %v\n", node.GlobalHallRequests)

	activeConnReq := make(map[int]messages.ConnectionReq)
	currentNodeHallAssignments := make(map[int][config.NUM_FLOORS][2]bool)
	hallAssignmentCounter := 0
	var nextNodeState nodestate

	// inform the global hall request transmitter of the new global hall requests
	node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
	node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)
	node.GlobalHallReqTransmitEnableTx <- true
	node.HallRequestAssignerTransmitEnableTx <- true

	select {
	case node.commandToServerTx <- "startConnectionTimeoutDetection":
		// Command sent successfully
	default:
		// Command not sent, channel is full
		fmt.Printf("Warning: Command channel is full, command %s not sent\n", "startConnectionTimeoutDetection")
	}

ForLoop:
	for {
	Select:
		select {
		case elevMsg := <-node.ElevatorEventRx:
			switch elevMsg.EventType {
			case singleelevator.ElevStatusUpdateEvent:
				fmt.Printf("Received elevator status update, stuck: %v\n", elevMsg.IsElevDown)
				// if the elevator is no longer functioning, we go to inactive
				if elevMsg.IsElevDown {
					nextNodeState = Inactive
					break ForLoop
				}
				break Select

			case singleelevator.HallButtonEvent:

				newHallReq := makeNewHallReq(node.ID, elevMsg)
				node.GlobalHallRequests = processNewHallRequest(node.GlobalHallRequests, newHallReq)
				node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
				select {
				case node.commandToServerTx <- "getActiveElevStates":
					// Command sent successfully
				default:
					// Command not sent, channel is full
					fmt.Printf("Warning: Command channel is full, command %s not sent\n", "getActiveElevStates")
				}
				fmt.Printf("Global hall requests: %v\n", node.GlobalHallRequests)
			}

		case myElevStates := <-node.MyElevStatesRx:
			// Transmit elevator states to network
			node.NodeElevStatesTx <- messages.NodeElevState{
				NodeID:    node.ID,
				ElevState: myElevStates,
			}

		case networkEvent := <-node.NetworkEventRx:
			fmt.Println("Network event received")
			switch networkEvent {
			case messagehandler.NodeHasLostConnection:

				fmt.Println("Connection timed out")
				nextNodeState = Disconnected
				break ForLoop

			case messagehandler.ActiveNodeCountChange:
				fmt.Println("Node connected or disconnected, starting redistribution of hall requests")
				select {
				case node.commandToServerTx <- "getActiveElevStates":
					// Command sent successfully
				default:
					// Command not sent, channel is full
					fmt.Printf("Warning: Command channel is full, command %s not sent\n", "getActiveElevStates")
				}
			}

		case newHallReq := <-node.NewHallReqRx:
			node.GlobalHallRequests = processNewHallRequest(node.GlobalHallRequests, newHallReq)
			node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
			select {
			case node.commandToServerTx <- "getActiveElevStates":
				// Command sent successfully
			default:
				// Command not sent, channel is full
				fmt.Printf("Warning: Command channel is full, command %s not sent\n", "getActiveElevStates")
			}
			fmt.Printf("Global hall requests: %v\n", node.GlobalHallRequests)

		case connReq := <-node.ConnectionReqRx:

			activeConnReq[connReq.NodeID] = connReq
			select {
			case node.commandToServerTx <- "getAllElevStates":
				// Command sent successfully
			default:
				// Command not sent, channel is full
				fmt.Printf("Warning: Command channel is full, command %s not sent\n", "getAllElevStates")
			}
		case elevStatesUpdate := <-node.NodeElevStateUpdate:

			switch elevStatesUpdate.DataType {

			case messagehandler.ActiveElevStates:
				fmt.Printf("Computing assignments:\n")
				// increase the hall assignment counter
				hallAssignmentCounter++

				// Guard clause to break out of the loop if there are no active connection requests
				if util.MapIsEmpty(elevStatesUpdate.NodeElevStatesMap) {
					break Select
				}
				computationResult := computeHallAssignments(node.ID,
					elevStatesUpdate,
					node.GlobalHallRequests,
					hallAssignmentCounter)

				node.ElevLightAndAssignmentUpdateTx <- computationResult.MyAssignment

				for _, hallAssignment := range computationResult.OtherAssignments {
					node.HallAssignmentTx <- hallAssignment
				}
				currentNodeHallAssignments = computationResult.NodeHallAssignments
				// Printing out each node's hall assignments
				// for id, hallAssignments := range currentNodeHallAssignments {
				// 	fmt.Printf("Node %d hall assignments: %v\n", id, hallAssignments)
				// }
				// fmt.Println("")

			case messagehandler.AllElevStates:
				infoToNodes := processConnectionRequestsFromOtherNodes(elevStatesUpdate, activeConnReq)
				for _, infoMessage := range infoToNodes.CabRequests {
					node.CabRequestInfoTx <- infoMessage
				}
			case messagehandler.HallAssignmentRemoved:
				fmt.Println("Hall assignment removed")
				node.GlobalHallRequests = updateGlobalHallRequests(currentNodeHallAssignments, elevStatesUpdate.NodeElevStatesMap, hallAssignmentCounter)
				fmt.Printf("Global hall requests: %v\n", node.GlobalHallRequests)

				node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
				node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)
			}

		case <-node.HallAssignmentsRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
			// when you get a message on any of these channels, do nothing
		}
	}

	// stop transmitters
	node.GlobalHallReqTransmitEnableTx <- false
	node.HallRequestAssignerTransmitEnableTx <- false
	select {
	case node.commandToServerTx <- "stopConnectionTimeoutDetection":
		// Command sent successfully
	default:
		// Command not sent, channel is full
		fmt.Printf("Warning: Command channel is full, command %s not sent\n", "stopConnectionTimeoutDetection")
	}
	node.TOLC = time.Now()
	fmt.Printf("Exiting master, setting TOLC to %v\n", node.TOLC)
	return nextNodeState

}

// HallAssignmentResult is a struct that holds the result of the hall assignment computation
type HallAssignmentResult struct {
	NodeHallAssignments map[int][config.NUM_FLOORS][2]bool
	MyAssignment        singleelevator.LightAndAssignmentUpdate
	OtherAssignments    map[int]messages.NewHallAssignments
}

type connectionRequestHandler struct {
	CabRequests map[int]messages.CabRequestInfo
}

func updateGlobalHallRequests(assignedNodeHallAssignments map[int][config.NUM_FLOORS][2]bool,
	recentNodeElevStates map[int]elevator.ElevatorState, globalHallRequests [config.NUM_FLOORS][2]bool, hallAssignmentCounter int) [config.NUM_FLOORS][2]bool {

	for id, hallAssignments := range assignedNodeHallAssignments {
		if nodeElevState, ok := recentNodeElevStates[id]; ok {

			// if the counter value is incorrect, we skip the node
			if nodeElevState.HACounterVersion != hallAssignmentCounter {
				continue
			}
			for floor := range config.NUM_FLOORS {
				for btn := range 2 {
					if hallAssignments[floor][btn] && !nodeElevState.MyHallAssignments[floor][btn] {
						globalHallRequests[floor][btn] = true
					}
				}
			}
		}
	}

	return globalHallRequests
}

func computeHallAssignments(
	myID int,
	elevStatesUpdate messagehandler.ElevStateUpdate,
	globalHallRequests [config.NUM_FLOORS][2]bool,
	HACounter int) HallAssignmentResult {

	var result HallAssignmentResult
	result.NodeHallAssignments = make(map[int][config.NUM_FLOORS][2]bool)
	// run the hall request assigner algorithm
	hraOutput := hallRequestAssigner.HRAalgorithm(elevStatesUpdate.NodeElevStatesMap, globalHallRequests)

	result.OtherAssignments = make(map[int]messages.NewHallAssignments)
	// fmt.Printf("Hall request assigner output: %v\n", hraOutput)
	// make the hall assignments for all nodes
	for id, hallRequests := range hraOutput {
		result.NodeHallAssignments[id] = hallRequests
		// if the assignment is for me, we make the light and assignment message
		if id == myID {
			result.MyAssignment = makeHallAssignmentAndLightMessage(hallRequests, globalHallRequests)
		} else {
			// if the assignment is for another node, we make a new hall assignment message
			result.OtherAssignments[id] = messages.NewHallAssignments{NodeID: id, HallAssignment: hallRequests, MessageID: 0, HallAssignmentCounter: HACounter}
		}
	}
	return result
}

func processConnectionRequestsFromOtherNodes(elevStatesUpdate messagehandler.ElevStateUpdate,
	activeConnReq map[int]messages.ConnectionReq) connectionRequestHandler {

	var result connectionRequestHandler
	fmt.Printf("Dealing with connreqs\n")
	// make cab request info for all nodes that have sent a connection request
	result.CabRequests = make(map[int]messages.CabRequestInfo)
	for id := range activeConnReq {
		var cabRequestInfo messages.CabRequestInfo
		// if  we have info about the node, we send it, otherwise we send an empty slice
		if states, ok := elevStatesUpdate.NodeElevStatesMap[id]; ok {
			cabRequestInfo = messages.CabRequestInfo{CabRequest: states.CabRequests, ReceiverNodeID: id}
		} else {
			cabRequestInfo = messages.CabRequestInfo{CabRequest: [config.NUM_FLOORS]bool{}, ReceiverNodeID: id}
		}
		// add the cab request info to the result
		result.CabRequests[id] = cabRequestInfo
	}
	return result

}

func processNewHallRequest(globalHallRequests [config.NUM_FLOORS][2]bool,
	newHallReq messages.NewHallReq) [config.NUM_FLOORS][2]bool {
	// if button is invalid we return false
	if newHallReq.HallReq.Button == elevator.ButtonCab {
		// fmt.Printf("Received a new hall request, but the button type was invalid\n")
		return globalHallRequests
	}
	// if the button is valid we update the global hall requests
	globalHallRequests[newHallReq.HallReq.Floor][newHallReq.HallReq.Button] = true
	return globalHallRequests
}

func anyHallRequestsActive(globalHallRequests [config.NUM_FLOORS][2]bool) bool {
	for floor := range config.NUM_FLOORS {
		for btn := range 2 {
			if globalHallRequests[floor][btn] {
				return true
			}
		}
	}
	return false
}
