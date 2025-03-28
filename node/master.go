package node

import (
	"elev/config"
	"elev/costFNS/hallrequestassigner"
	"elev/elevator"
	"elev/network/communication"
	"elev/network/messages"
	"elev/singleelevator"
	"elev/util"
	"fmt"
	"time"
)

// HRAresult is a struct that holds the result of the hall assignment computation
type HRAresult struct {
	NodeHallAssignments map[int][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool
	MyAssignment        singleelevator.LightAndAssignmentUpdate
	OtherAssignments    map[int]messages.NewHallAssignments
}

func MasterProgram(node *NodeData) nodestate {
	fmt.Printf("Node %v is now Master\n", node.ID)

	// Initialize master state: initialize map for connection requests from other nodes, hall assignments, and broadcast ticker.
	activeConnReq := make(map[int]messages.ConnectionReq)
	HallAssignmentsPerNodeMap := make(map[int][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool)
	hallAssignmentCounter := 0
	var nextNodeState nodestate
	GlobalHallReqSendTicker := time.NewTicker(config.MASTER_BROADCAST_INTERVAL)

	// inform your own elevator of the lights
	node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)

	// enable the hall assignment transmitter
	node.HallAssignmentTransmitterEnableTx <- true

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
			// Checks if the elevator for master node is down
			case singleelevator.ElevStatusUpdateEvent:
				if elevMsg.ElevIsDown {
					nextNodeState = Inactive
					break ForLoop
				}
				break Select

			case singleelevator.HallButtonEvent:
				// If master recieves new hall request from its own elevator update global hall requests
				newHallReq := makeNewHallReq(node.ID, elevMsg)
				node.GlobalHallRequests = addNewHallRequest(node.GlobalHallRequests, newHallReq)

				select {
				case node.commandToServerTx <- "getActiveElevStates":
				default:
					fmt.Printf("Warning: Command channel is full, command %s not sent\n", "getActiveElevStates")
				}

			}

		case myElevStates := <-node.MyElevStatesRx:
			// Master recieves the state of its own elevator and sends to server
			node.NodeElevStatesTx <- messages.NodeElevState{
				NodeID:    node.ID,
				ElevState: myElevStates,
			}

		case networkEvent := <-node.NetworkEventRx:
			switch networkEvent {
			case communication.NodeHasLostConnection: // If master has lost connection go disconnected

				nextNodeState = Disconnected
				break ForLoop

			case communication.ActiveNodeCountChange: // If a node has connected or disconnected to network redistribute hall requests

				select {
				// request the active elevator states from the server, to run the hall assignment algorithm
				case node.commandToServerTx <- "getActiveElevStates":
				default:
					fmt.Printf("Warning: Command channel is full, command %s not sent\n", "getActiveElevStates")
				}
			}

		case newHallReq := <-node.NewHallReqRx:
			node.GlobalHallRequests = addNewHallRequest(node.GlobalHallRequests, newHallReq)

			// request the active elevator states from the server, to run the hall assignment algorithm
			select {
			case node.commandToServerTx <- "getActiveElevStates":
			default:
				fmt.Printf("Warning: Command channel is full, command %s not sent\n", "getActiveElevStates")
			}

		case connReq := <-node.ConnectionReqRx:
			// save the new connection request, and request all elev states from the server, so we can check if we know any of the cab button presses of the elevator that wants to connect
			activeConnReq[connReq.NodeID] = connReq

			select {
			case node.commandToServerTx <- "getAllElevStates":
			default:
				fmt.Printf("Warning: Command channel is full, command %s not sent\n", "getAllElevStates")
			}

		case elevStatesUpdate := <-node.ElevStateUpdatesFromServer:

			switch elevStatesUpdate.DataType {

			case communication.ActiveElevStates:

				// Guard clause to break out of the loop if there are no active nodes
				if util.MapIsEmpty(elevStatesUpdate.NodeElevStatesMap) {
					break Select
				}

				// increment the hall assignment counter
				hallAssignmentCounter = util.IncrementIntCounter(hallAssignmentCounter)

				// Run HRA algorithm to distribute hall requests
				computationResult := computeHallAssignments(node.ID,
					elevStatesUpdate,
					node.GlobalHallRequests,
					hallAssignmentCounter)

				// update my elevator with the new assignments
				node.ElevLightAndAssignmentUpdateTx <- computationResult.MyAssignment

				// inform the hall assignment transmitter that there are new assignments
				for _, hallAssignment := range computationResult.OtherAssignments {
					node.HallAssignmentTx <- hallAssignment
				}
				// remember which node does what, used for clearing hall assignments
				HallAssignmentsPerNodeMap = computationResult.NodeHallAssignments

			case communication.AllElevStates: // Recieved all states of all known nodes to check if we have info about the nodes wanting to connect to the network
				infoToNodes := makeConnectionRequestReplies(elevStatesUpdate, activeConnReq)
				for _, infoMessage := range infoToNodes {
					node.CabRequestInfoTx <- infoMessage
				}
			case communication.HallAssignmentRemoved:
				node.GlobalHallRequests = updateGlobalHallRequests(HallAssignmentsPerNodeMap, elevStatesUpdate.NodeElevStatesMap, node.GlobalHallRequests, hallAssignmentCounter)
				node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)
			}

		case <-GlobalHallReqSendTicker.C:
			// Periodically broadcast global hall requests
			node.ContactCounter = util.IncrementUint64Counter(node.ContactCounter)
			node.GlobalHallRequestTx <- makeGlobalHallRequestMessage(node.GlobalHallRequests, node.ContactCounter)

		case <-node.HallAssignmentsRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
			// read these to prevent blocking
		}
	}

	// stop transmitter
	node.HallAssignmentTransmitterEnableTx <- false

	select {
	case node.commandToServerTx <- "stopConnectionTimeoutDetection":
	default:
		fmt.Printf("Warning: Command channel is full, command %s not sent\n", "stopConnectionTimeoutDetection")
	}

	fmt.Printf("Exiting master, counter value is %v\n", node.ContactCounter)
	return nextNodeState

}
func makeGlobalHallRequestMessage(
	globalHallRequests [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool,
	counterValue uint64) messages.GlobalHallRequest {
	return messages.GlobalHallRequest{HallRequests: globalHallRequests,
		CounterValue: counterValue}
}

// go through all the active hall assignments of all node and check if any hall assignment has been completed
func updateGlobalHallRequests(
	nodeHallAssignments map[int][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool,
	recentNodeElevStates map[int]elevator.ElevatorStateReport,
	globalHallRequests [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool,
	hallAssignmentCounter int) [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool {

	// looping through all the nodes and their respective hall assignments
	for id, hallAssignments := range nodeHallAssignments {
		// if we have info about the node
		if nodeElevState, ok := recentNodeElevStates[id]; ok {

			// if the counter value is incorrect, we skip the node
			if nodeElevState.HACounterVersion != hallAssignmentCounter {
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
	elevStatesUpdate communication.ElevStateUpdate,
	globalHallRequests [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool,
	HACounter int) HRAresult {

	var result HRAresult
	result.NodeHallAssignments = make(map[int][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool)
	hraOutput := hallrequestassigner.HRAalgorithm(elevStatesUpdate.NodeElevStatesMap, globalHallRequests)

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

func makeConnectionRequestReplies(elevStatesUpdate communication.ElevStateUpdate,
	activeConnReq map[int]messages.ConnectionReq) map[int]messages.CabRequestInfo {

	// make cab request info for all nodes that have sent a connection request
	result := make(map[int]messages.CabRequestInfo)
	for id := range activeConnReq {
		var cabRequestInfo messages.CabRequestInfo

		// if  we have info about the node, we use it, otherwise we just send an array of all false
		if states, ok := elevStatesUpdate.NodeElevStatesMap[id]; ok {
			cabRequestInfo = messages.CabRequestInfo{CabRequest: states.CabRequests, ReceiverNodeID: id}
		} else {
			cabRequestInfo = messages.CabRequestInfo{CabRequest: [config.NUM_FLOORS]bool{}, ReceiverNodeID: id}
		}
		result[id] = cabRequestInfo
	}
	return result

}

// add a new button press to the global hall assignments and return it
func addNewHallRequest(globalHallRequests [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool,
	newHallReq messages.NewHallReq) [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool {

	if newHallReq.HallReq.Button == elevator.ButtonCab {
		return globalHallRequests
	}
	// if the button is valid we update the global hall requests
	globalHallRequests[newHallReq.HallReq.Floor][newHallReq.HallReq.Button] = true
	return globalHallRequests
}
