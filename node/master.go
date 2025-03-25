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
	fmt.Printf("Node %d is now a Master\n", node.ID)

	oldGlobalHallRequests := node.GlobalHallRequests
	// Check if we should distribute hall requests
	shouldDistributeHallRequests := false
	for floor := 0; floor < config.NUM_FLOORS; floor++ {
		for btn := 0; btn < 2; btn++ {
			if node.GlobalHallRequests[floor][btn] {
				fmt.Printf("Global hall requests: %v\n", node.GlobalHallRequests)
				shouldDistributeHallRequests = true
				node.commandToServerTx <- "getActiveElevStates"
				break
			}
		}
		if shouldDistributeHallRequests {
			break
		}
	}

	activeConnReq := make(map[int]messages.ConnectionReq)

	var nextNodeState nodestate

	// inform the global hall request transmitter of the new global hall requests
	fmt.Printf("Initiating master: Global requests: %v\n", node.GlobalHallRequests)
	node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
	node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)

	// start the transmitters
	node.GlobalHallReqTransmitEnableTx <- true
	node.HallRequestAssignerTransmitEnableTx <- true
	node.commandToServerTx <- "startConnectionTimeoutDetection"

ForLoop:
	for {
	Select:
		select {
		case elevMsg := <-node.ElevatorEventRx:
			switch elevMsg.EventType {

			case singleelevator.DoorStuckEvent:
				fmt.Println("DoorStuckEvent")
				// if the door is stuck, we go to inactive
				if elevMsg.DoorIsStuck {
					nextNodeState = Inactive
					break ForLoop
				}

				break Select
			}

		case <- time.Tick(500 * time.Millisecond):
			node.commandToServerTx <- "getActiveElevStates"

		
		case connReq := <-node.ConnectionReqRx:
			if connReq.NodeID != node.ID {
				activeConnReq[connReq.NodeID] = connReq
				node.commandToServerTx <- "getAllElevStates"
			}
		case elevStatesUpdate := <-node.NodeElevStateUpdate:
			if!(elevStatesUpdate.OnlyActiveNodes){
				infoToNodes := processConnectionRequestsFromOtherNodes(elevStatesUpdate, activeConnReq)
				for _, info := range infoToNodes.CabRequests {
					node.CabRequestInfoTx <- info
				}
			}else if(elevStatesUpdate.OnlyActiveNodes){
				node.GlobalHallRequests = mergeGlobalHallRequests(elevStatesUpdate.NodeElevStatesMap)
				orderAdded, orderRemoved := checkGlobalHallRequestsChange(oldGlobalHallRequests, node.GlobalHallRequests) 
					
				if(orderAdded){
				computationResult, shouldDistribute := 
					computeHallAssignments(shouldDistributeHallRequests, 
											node.ID, 
											elevStatesUpdate, 
											node.GlobalHallRequests)
					// update distribution flag
					shouldDistributeHallRequests = shouldDistribute
					// send the hall assignments to the hall assignment transmitter
					for _, newHallAssignment := range computationResult.OtherAssignments {
						node.HallAssignmentTx <- newHallAssignment
					}
					// send hallAssignment to the my elevator
					node.ElevLightAndAssignmentUpdateTx <- computationResult.MyAssignment
					oldGlobalHallRequests = node.GlobalHallRequests
				}
				if(orderRemoved){
					node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)
					node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
					oldGlobalHallRequests = node.GlobalHallRequests
				}

			}

		case networkEvent := <-node.NetworkEventRx:
			fmt.Println("Network event received")

			if networkEvent == messagehandler.NodeHasLostConnection {
				fmt.Println("Connection timed out")
				nextNodeState = Disconnected
				break ForLoop

			} else if networkEvent == messagehandler.NodeConnectDisconnect {
				fmt.Println("Node connected or disconnected, starting redistribution of hall requests")
				shouldDistributeHallRequests = true
				node.commandToServerTx <- "getActiveElevStates"
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
	node.commandToServerTx <- "stopConnectionTimeoutDetection"
	node.TOLC = time.Now()
	fmt.Printf("Exiting master, setting TOLC to %v\n", node.TOLC)
	return nextNodeState
}

// HallAssignmentResult is a struct that holds the result of the hall assignment computation
type HallAssignmentResult struct {
	MyAssignment      singleelevator.LightAndAssignmentUpdate
	OtherAssignments  map[int]messages.NewHallAssignments
	GlobalHallRequest messages.GlobalHallRequest
}

type connectionRequestHandler struct {
	CabRequests map[int]messages.CabRequestInfo
}

func computeHallAssignments(shouldDistribute bool,
	myID int,
	elevStatesUpdate messagehandler.ElevStateUpdate,
	globalHallRequests [config.NUM_FLOORS][2]bool) (HallAssignmentResult, bool) {
	var result HallAssignmentResult
	// if we should distribute, we run the hall request assigner algorithm
		// run the hall request assigner algorithm
	if shouldDistribute {
		hraOutput := hallRequestAssigner.HRAalgorithm(elevStatesUpdate.NodeElevStatesMap, globalHallRequests)
		fmt.Printf("Hall request assigner output: %v\n", hraOutput)
		result.OtherAssignments = make(map[int]messages.NewHallAssignments)
		// fmt.Printf("Hall request assigner output: %v\n", hraOutput)
		// make the hall assignments for all nodes
		for id, hallRequests := range hraOutput {
			// if the assignment is for me, we make the light and assignment message
			if id == myID {
				result.MyAssignment = makeHallAssignmentAndLightMessage(hallRequests, globalHallRequests)
			} else { // if the assignment is for another node, we make a new hall assignment message
				result.OtherAssignments[id] = messages.NewHallAssignments{NodeID: id, HallAssignment: hallRequests, MessageID: 0}
			}
		}
	}
	// make the global hall request message
	result.GlobalHallRequest = messages.GlobalHallRequest{HallRequests: globalHallRequests}
	// turn of shouldDistribute flag
	shouldDistribute = false
	// if we get all nodes we make cab request info for connreq nodes
	return result, shouldDistribute
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

	func checkGlobalHallRequestsChange(oldGlobalHallRequests [config.NUM_FLOORS][2]bool,
		newGlobalHallReq [config.NUM_FLOORS][2]bool) (orderAdded bool, orderRemoved bool) {
		for floor := 0; floor < config.NUM_FLOORS; floor++ {
			for button := 0; button < 2; button++ {
				// Check if change is from (false -> true), assignment added
				if !oldGlobalHallRequests[floor][button] && newGlobalHallReq[floor][button] {
					orderAdded = true
				}
				// Check if change is from (true -> false), assignment complete
				if oldGlobalHallRequests[floor][button] && !newGlobalHallReq[floor][button] {
					orderRemoved = true
				}
			}
		}
		return orderAdded, orderRemoved
	}
	

func mergeGlobalHallRequests(allStates map[int]elevator.ElevatorState) [config.NUM_FLOORS][2]bool {
	var globalHallRequests [config.NUM_FLOORS][2]bool

	for _, state := range allStates {
		for floor := 0; floor < config.NUM_FLOORS; floor++ {
			for button := 0; button < 2; button++ {
				if state.LocalHallRequests[floor][button] {
					globalHallRequests[floor][button] = true
				}
			}
		}
	}

	return globalHallRequests
}



// func ProcessHAComplete(
// 	globalHallRequests [config.NUM_FLOORS][2]bool,
// 	buffer MessageIDBuffer,
// 	ha messages.HallAssignmentComplete) ([config.NUM_FLOORS][2]bool, MessageIDBuffer, bool) {
// 	updateNeeded := false
// 	// Check if the message is new
// 	if !buffer.Contains(ha.MessageID) {
// 		// the message is new, we update the global hall requests
// 		if ha.HallButton != elevator.ButtonCab {
// 			globalHallRequests[ha.Floor][ha.HallButton] = false
// 		}
// 		// we add the message to the buffer
// 		buffer.Add(ha.MessageID)
// 		// we set the update flag to true
// 		updateNeeded = true
// 	}
// 	// return the updated global hall requests, the updated buffer and the update flag
// 	return globalHallRequests, buffer, updateNeeded
// }

// func ProcessNewHallRequest(
// 	globalHallRequests [config.NUM_FLOORS][2]bool,
// 	newHallReq messages.NewHallRequest) ([config.NUM_FLOORS][2]bool, bool) {
// 	// if button is invalid we return false
// 	if newHallReq.HallButton == elevator.ButtonCab {
// 		// fmt.Printf("Received a new hall request, but the button type was invalid\n")
// 		return globalHallRequests, false
// 	}
// 	// if the button is valid we update the global hall requests
// 	globalHallRequests[newHallReq.Floor][newHallReq.HallButton] = true
// 	return globalHallRequests, true
// }
