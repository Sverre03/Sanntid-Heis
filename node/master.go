package node

import (
	"elev/Network/messagehandler"
	"elev/Network/messages"
	"elev/config"
	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"elev/singleelevator"
	"fmt"
	"time"
)

func MasterProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now Master\n", node.ID)

	activeConnReq := make(map[int]messages.ConnectionReq)
	currentNodeHallAssignments := make(map[int][config.NUM_FLOORS][2]bool)
	var nextNodeState nodestate

	//Check if we should distribute hall requests
	if shouldDistributeHallRequests(node.GlobalHallRequests) {
		sendCommandToServer("getActiveElevStates", node)
	}

	// inform the global hall request transmitter of the new global hall requests
	fmt.Printf("Initiating master: Global requests: %v\n", node.GlobalHallRequests)
	node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
	node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)

	// start the transmitters
	node.GlobalHallReqTransmitEnableTx <- true
	node.HallRequestAssignerTransmitEnableTx <- true
	sendCommandToServer("startConnectionTimeoutDetection", node)

ForLoop:
	for {
	Select:
		select {
		case elevMsg := <-node.ElevatorEventRx:
			switch elevMsg.EventType {
			case singleelevator.DoorStuckEvent:
				fmt.Printf("Master received door stuck event: stuck: %v\n", elevMsg.DoorIsStuck)
				// if the door is stuck, we go to inactive
				if doorIsStuck(elevMsg) {
					nextNodeState = Inactive
					break ForLoop
				}
				break Select

			case singleelevator.HallButtonEvent:

				newHallReq := makeNewHallReq(node.ID, elevMsg)
				node.GlobalHallRequests = processNewHallRequest(node.GlobalHallRequests, newHallReq)
				node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
				sendCommandToServer("getActiveElevStates", node)
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
			
			case messagehandler.NodeConnectDisconnect:
				fmt.Println("Node connected or disconnected, starting redistribution of hall requests")
				sendCommandToServer("getActiveElevStates", node)
			}
			

		case newHallReq := <-node.NewHallReqRx:
			node.GlobalHallRequests = processNewHallRequest(node.GlobalHallRequests, newHallReq)
			node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
			sendCommandToServer("getActiveElevStates", node)
			fmt.Printf("Global hall requests: %v\n", node.GlobalHallRequests)

		case connReq := <-node.ConnectionReqRx:

			activeConnReq[connReq.NodeID] = connReq
			sendCommandToServer("getAllElevStates", node)

		case elevStatesUpdate := <-node.NodeElevStateUpdate:

			switch elevStatesUpdate.DataType {

			case messagehandler.ActiveElevStates:
				fmt.Printf("Computing assignments:\n")
				// Guard clause to break out of the loop if there are no active connection requests
				if mapIsEmpty(elevStatesUpdate.NodeElevStatesMap) {
					break Select
				}
				computationResult := computeHallAssignments(node.ID,
					elevStatesUpdate,
					node.GlobalHallRequests)

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
				for _, info := range infoToNodes.CabRequests {
					node.CabRequestInfoTx <- info
				}
			case messagehandler.HallAssignmentRemoved:
				// fmt.Println("Hall assignment removed")
				// fmt.Printf("Updated elevator states: %v\n", elevStatesUpdate.NodeElevStatesMap)
				node.GlobalHallRequests = updateGlobalHallRequests(currentNodeHallAssignments, elevStatesUpdate.NodeElevStatesMap)
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
	sendCommandToServer("stopConnectionTimeoutDetection", node)
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
	recentNodeElevStates map[int]elevator.ElevatorState) [config.NUM_FLOORS][2]bool {

	var globalHallRequests [config.NUM_FLOORS][2]bool

	for floor := range config.NUM_FLOORS {
		for btn := range 2 {
			globalHallRequests[floor][btn] = false
		}
	}

	// If the hall assignment which was assigned to a node is still active for that node, we add it to the global hall requests
	for id, hallAssignments := range assignedNodeHallAssignments {
		if nodeElevState, ok := recentNodeElevStates[id]; ok {
			for floor := range config.NUM_FLOORS {
				for btn := range 2 {
					if hallAssignments[floor][btn] && nodeElevState.MyHallAssignments[floor][btn] {
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
	globalHallRequests [config.NUM_FLOORS][2]bool) HallAssignmentResult {

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
			result.OtherAssignments[id] = messages.NewHallAssignments{NodeID: id, HallAssignment: hallRequests, MessageID: 0}
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
			emptySlice := [config.NUM_FLOORS]bool{}
			cabRequestInfo = messages.CabRequestInfo{CabRequest: emptySlice, ReceiverNodeID: id}
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
	fmt.Printf("updating global hall requests\n")
	globalHallRequests[newHallReq.HallReq.Floor][newHallReq.HallReq.Button] = true
	return globalHallRequests
}

func shouldDistributeHallRequests(globalHallRequests [config.NUM_FLOORS][2]bool) bool {
	for floor := range config.NUM_FLOORS {
		for btn := range 2 {
			if globalHallRequests[floor][btn] {
				return true
			}
		}
	}
	return false
}
