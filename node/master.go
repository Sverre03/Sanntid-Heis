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

// HallAssignmentResult is a struct that holds the result of the hall assignment computation
type HallAssignmentResult struct {
	NodeHallAssignments map[int][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool
	MyAssignment        singleelevator.LightAndAssignmentUpdate
	OtherAssignments    map[int]messages.NewHallAssignments
}

type connectionRequestHandler struct {
	CabRequests map[int]messages.CabRequestInfo
}

func MasterProgram(node *NodeData) nodestate {
	fmt.Printf("Initiating master: Global requests: %v\n", node.GlobalHallRequests)

	activeConnReq := make(map[int]messages.ConnectionReq)
	nodeHallAssignments := make(map[int][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool)
	hallAssignmentCounter := 0
	var nextNodeState nodestate

	GlobalHallReqSendTicker := time.NewTicker(config.MASTER_BROADCAST_INTERVAL)
	// inform the global hall request transmitter of the new global hall requests
	node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)

	node.HallRequestAssignerTransmitEnableTx <- true

	select {
	case node.commandToServerTx <- "startConnectionTimeoutDetection":
	default:
		fmt.Printf("Warning: Command channel is full, command %s not sent\n", "startConnectionTimeoutDetection")
	}

ForLoop:
	for {
	Select:
		select {
		case elevMsg := <-node.ElevatorEventRx:
			switch elevMsg.EventType {
			case singleelevator.ElevStatusUpdateEvent:
				if elevMsg.ElevIsDown {
					nextNodeState = Inactive
					break ForLoop
				}
				break Select

			case singleelevator.HallButtonEvent:

				newHallReq := makeNewHallReq(node.ID, elevMsg)
				node.GlobalHallRequests = processNewHallRequest(node.GlobalHallRequests, newHallReq)

				select {
				case node.commandToServerTx <- "getActiveElevStates":
				default:
					fmt.Printf("Warning: Command channel is full, command %s not sent\n", "getActiveElevStates")
				}

				fmt.Printf("Global hall requests: %v\n", node.GlobalHallRequests)
			}

		case myElevStates := <-node.MyElevStatesRx:
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
				default:
					fmt.Printf("Warning: Command channel is full, command %s not sent\n", "getActiveElevStates")
				}
			}

		case newHallReq := <-node.NewHallReqRx:
			node.GlobalHallRequests = processNewHallRequest(node.GlobalHallRequests, newHallReq)

			select {
			case node.commandToServerTx <- "getActiveElevStates":
			default:
				fmt.Printf("Warning: Command channel is full, command %s not sent\n", "getActiveElevStates")
			}

			fmt.Printf("Global hall requests: %v\n", node.GlobalHallRequests)

		case connReq := <-node.ConnectionReqRx:
			activeConnReq[connReq.NodeID] = connReq

			select {
			case node.commandToServerTx <- "getAllElevStates":
			default:
				fmt.Printf("Warning: Command channel is full, command %s not sent\n", "getAllElevStates")
			}

		case elevStatesUpdate := <-node.NodeElevStateUpdate:

			switch elevStatesUpdate.DataType {

			case messagehandler.ActiveElevStates:

				// Guard clause to break out of the loop if there are no active nodes
				if util.MapIsEmpty(elevStatesUpdate.NodeElevStatesMap) {
					break Select
				}

				// increment the hall assignment counter
				hallAssignmentCounter = util.IncrementIntCounter(hallAssignmentCounter)

				fmt.Printf("Computing assignments: counter value is now %d\n", hallAssignmentCounter)

				computationResult := computeHallAssignments(node.ID,
					elevStatesUpdate,
					node.GlobalHallRequests,
					hallAssignmentCounter)

				node.ElevLightAndAssignmentUpdateTx <- computationResult.MyAssignment

				for _, hallAssignment := range computationResult.OtherAssignments {
					node.HallAssignmentTx <- hallAssignment
				}
				nodeHallAssignments = computationResult.NodeHallAssignments

			case messagehandler.AllElevStates:
				infoToNodes := processConnectionRequestsFromOtherNodes(elevStatesUpdate, activeConnReq)
				for _, infoMessage := range infoToNodes.CabRequests {
					node.CabRequestInfoTx <- infoMessage
				}
			case messagehandler.HallAssignmentRemoved:
				node.GlobalHallRequests = updateGlobalHallRequests(nodeHallAssignments, elevStatesUpdate.NodeElevStatesMap, node.GlobalHallRequests, hallAssignmentCounter)
				node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)
			}

		case <-GlobalHallReqSendTicker.C:
			node.ContactCounter++
			node.GlobalHallRequestTx <- makeGlobalHallRequestMessage(node.GlobalHallRequests, node.ContactCounter)

		case <-node.HallAssignmentsRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
		}
	}

	// stop transmitters
	node.HallRequestAssignerTransmitEnableTx <- false

	select {
	case node.commandToServerTx <- "stopConnectionTimeoutDetection":
	default:
		fmt.Printf("Warning: Command channel is full, command %s not sent\n", "stopConnectionTimeoutDetection")
	}

	fmt.Printf("Exiting master, counter value is %v\n", node.ContactCounter)
	return nextNodeState

}
func makeGlobalHallRequestMessage(globalHallRequests [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool,
	counterValue uint64) messages.GlobalHallRequest {
	return messages.GlobalHallRequest{HallRequests: globalHallRequests,
		CounterValue: counterValue}
}

func updateGlobalHallRequests(
	nodeHallAssignments map[int][config.NUM_FLOORS][2]bool,
	recentNodeElevStates map[int]elevator.ElevatorState,
	globalHallRequests [config.NUM_FLOORS][2]bool,
	hallAssignmentCounter int) [config.NUM_FLOORS][2]bool {

	// loop through all the nodes and their respective hall assignments
	for id, hallAssignments := range nodeHallAssignments {
		// if we have info about the node
		if nodeElevState, ok := recentNodeElevStates[id]; ok {

			// if the counter value is incorrect, we skip the node
			if nodeElevState.HACounterVersion != hallAssignmentCounter {
				//fmt.Printf("Skipping node %d, counter value incorrect\n", id)
				continue
			}
			for floor := range config.NUM_FLOORS {
				for btn := range config.NUM_HALL_BUTTONS {
					if hallAssignments[floor][btn] && !nodeElevState.MyHallAssignments[floor][btn] {
						// if the hall assignment is active and the node does not have it, we remove it
						globalHallRequests[floor][btn] = false
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
	globalHallRequests [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool,
	HACounter int) HallAssignmentResult {

	var result HallAssignmentResult
	result.NodeHallAssignments = make(map[int][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool)
	hraOutput := hallRequestAssigner.HRAalgorithm(elevStatesUpdate.NodeElevStatesMap, globalHallRequests)

	result.OtherAssignments = make(map[int]messages.NewHallAssignments)

	for id, hallRequests := range hraOutput {
		result.NodeHallAssignments[id] = hallRequests
		// if the assignment is for me, we make the light and assignment message
		if id == myID {
			result.MyAssignment = makeHallAssignmentAndLightMessage(hallRequests, globalHallRequests, HACounter)
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

func processNewHallRequest(globalHallRequests [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool,
	newHallReq messages.NewHallReq) [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool {
	// if button is invalid we return false
	if newHallReq.HallReq.Button == elevator.ButtonCab {
		return globalHallRequests
	}
	// if the button is valid we update the global hall requests
	globalHallRequests[newHallReq.HallReq.Floor][newHallReq.HallReq.Button] = true
	return globalHallRequests
}
