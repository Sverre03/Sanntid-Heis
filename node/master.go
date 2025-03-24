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

const bufferSize = 5

// A buffer that holds the last #buffersize message ids
type MessageIDBuffer struct {
	messageIDs [bufferSize]uint64
	size       int
	index      int
}

func makeNewMessageIDBuffer(bufferSize int) MessageIDBuffer {
	return MessageIDBuffer{size: bufferSize, index: 0}
}

// using Add, you can add a message ID to the buffer. It overwrites in a FIFO manner
func (buf *MessageIDBuffer) Add(id uint64) {
	if buf.index == buf.size-1 {
		buf.index = 0
	}
	buf.messageIDs[buf.index] = id
	buf.index += 1
}

// check if a message id is in the buffer
func (buf *MessageIDBuffer) Contains(id uint64) bool {
	for i := 0; i < buf.size; i++ {
		if buf.messageIDs[i] == id {
			return true
		}
	}
	return false
}

func MasterProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now a Master\n", node.ID)

	var myElevState messages.NodeElevState

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

	recentHACompleteBuffer := makeNewMessageIDBuffer(bufferSize)
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

			case singleelevator.HallButtonEvent:
				fmt.Printf("HallButtonEvent\n")
				// new hallbuttonpress from my elevator
				if elevMsg.ButtonEvent.Button != elevator.ButtonCab {
					node.GlobalHallRequests[elevMsg.ButtonEvent.Floor][elevMsg.ButtonEvent.Button] = true
					shouldDistributeHallRequests = true
				}

			case singleelevator.LocalHallAssignmentCompleteEvent:
				fmt.Println("LocalHallAssignmentCompleteEvent")
				// update the global hall assignments
				if elevMsg.ButtonEvent.Button != elevator.ButtonCab {
					node.GlobalHallRequests[elevMsg.ButtonEvent.Floor][elevMsg.ButtonEvent.Button] = false
				}
			}

			if shouldDistributeHallRequests {
				fmt.Printf("Global hall requests after Elevator Event: %v, event: %v\n", node.GlobalHallRequests, elevMsg)
				// fmt.Printf("New Global hall requests: %v\n", node.GlobalHallRequests)
				node.commandToServerTx <- "getActiveElevStates"
			}
			fmt.Printf("Global hall requests after Elevator Event: %v, event: %v\n", node.GlobalHallRequests, elevMsg)
			// update the hall request transmitter with the newest requests
			node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}

			node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)

		case myStates := <-node.MyElevStatesRx:
			// transmit elevator states to network
			myElevState = messages.NodeElevState{NodeID: node.ID, ElevState: myStates}
			node.NodeElevStatesTx <- myElevState

		case newHallReq := <-node.NewHallReqRx:

			updatedState, shouldDistribute := ProcessNewHallRequest(node.GlobalHallRequests, newHallReq)
			shouldDistributeHallRequests = shouldDistribute

			//if button is invalid we do nothing
			if !shouldDistribute {
				fmt.Println("Received a new hall request, but the button type was invalid")
				break Select
			}

			// update the global hall requests
			node.GlobalHallRequests = updatedState
			// fmt.Printf("New Global hall requests: %v\n", node.GlobalHallRequests)

			// send the global hall requests to the server for broadcast to update other nodes
			fmt.Printf("New global hall requests: %v\n", node.GlobalHallRequests)
			node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
			node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)
			// run getActiveElevStates to distribute the new hall requests
			node.commandToServerTx <- "getActiveElevStates"

		case elevStatesUpdate := <-node.NodeElevStateUpdate:
			fmt.Printf("Received new elevator states update: %v\n", elevStatesUpdate)
			// compute the hall assignments
			result, newShouldDistribute := ComputeHallAssignments(shouldDistributeHallRequests,
				elevStatesUpdate,
				myElevState,
				node.GlobalHallRequests,
				activeConnReq)
			// send the hall assignments to the hall assignment transmitter
			node.ElevLightAndAssignmentUpdateTx <- result.MyAssignment

			for _, assignment := range result.OtherAssignments {
				node.HallAssignmentTx <- assignment
			}

			// send the global hall requests to the server for broadcast to update other nodes

			node.GlobalHallRequestTx <- result.GlobalHallRequest

			node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)

			shouldDistributeHallRequests = newShouldDistribute

			for _, cabReqConnReqAnswer := range result.CabRequests {
				node.CabRequestInfoTx <- cabReqConnReqAnswer
				delete(activeConnReq, cabReqConnReqAnswer.ReceiverNodeID)
			}

		case connReq := <-node.ConnectionReqRx:
			if connReq.NodeID != node.ID {
				activeConnReq[connReq.NodeID] = connReq
				node.commandToServerTx <- "getAllElevStates"
			}

		case HA := <-node.HallAssignmentCompleteRx:
			// flag for updating the global hall requests and lights
			var updateNeeded bool
			node.GlobalHallRequests, recentHACompleteBuffer, updateNeeded =
				ProcessHAComplete(node.GlobalHallRequests, recentHACompleteBuffer, HA)

			if updateNeeded {
				fmt.Println("Received new hall assignment complete message")
				fmt.Printf("Global hall requests after completion: %v\n", node.GlobalHallRequests)
				// send the global hall requests to the server for broadcast to update other nodes
				node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}

				// send light update to elevator
				node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(node.GlobalHallRequests)
			}
			// send ack to the server
			// fmt.Printf("Acking complete message with id %d\n", HA.MessageID)
			node.AckTx <- messages.Ack{MessageID: HA.MessageID, NodeID: node.ID}

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
	CabRequests       map[int]messages.CabRequestInfo
}

func ComputeHallAssignments(shouldDistribute bool,
	elevStatesUpdate messagehandler.ElevStateUpdate,
	myElevState messages.NodeElevState,
	globalHallRequests [config.NUM_FLOORS][2]bool,
	activeConnReq map[int]messages.ConnectionReq) (HallAssignmentResult, bool) {
	var result HallAssignmentResult
	// if we should distribute, we run the hall request assigner algorithm
	if shouldDistribute && elevStatesUpdate.OnlyActiveNodes {
		// run the hall request assigner algorithm
		elevStatesUpdate.NodeElevStatesMap[myElevState.NodeID] = myElevState.ElevState
		hraOutput := hallRequestAssigner.HRAalgorithm(elevStatesUpdate.NodeElevStatesMap, globalHallRequests)
		fmt.Printf("Hall request assigner output: %v\n", hraOutput)
		result.OtherAssignments = make(map[int]messages.NewHallAssignments)
		// fmt.Printf("Hall request assigner output: %v\n", hraOutput)
		// make the hall assignments for all nodes
		for id, hallRequests := range hraOutput {
			// if the assignment is for me, we make the light and assignment message
			if id == myElevState.NodeID {
				result.MyAssignment = makeHallAssignmentAndLightMessage(hallRequests, globalHallRequests)
			} else { // if the assignment is for another node, we make a new hall assignment message
				result.OtherAssignments[id] = messages.NewHallAssignments{NodeID: id, HallAssignment: hallRequests, MessageID: 0}
			}
		}
		// make the global hall request message
		result.GlobalHallRequest = messages.GlobalHallRequest{HallRequests: globalHallRequests}
		// turn of shouldDistribute flag
		shouldDistribute = false
	}
	// if we get all nodes we make cab request info for connreq nodes
	if !elevStatesUpdate.OnlyActiveNodes {
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
	}
	return result, shouldDistribute
}

func ProcessHAComplete(
	globalHallRequests [config.NUM_FLOORS][2]bool,
	buffer MessageIDBuffer,
	ha messages.HallAssignmentComplete) ([config.NUM_FLOORS][2]bool, MessageIDBuffer, bool) {
	updateNeeded := false
	// Check if the message is new
	if !buffer.Contains(ha.MessageID) {
		// the message is new, we update the global hall requests
		if ha.HallButton != elevator.ButtonCab {
			globalHallRequests[ha.Floor][ha.HallButton] = false
		}
		// we add the message to the buffer
		buffer.Add(ha.MessageID)
		// we set the update flag to true
		updateNeeded = true
	}
	// return the updated global hall requests, the updated buffer and the update flag
	return globalHallRequests, buffer, updateNeeded
}

func ProcessNewHallRequest(
	globalHallRequests [config.NUM_FLOORS][2]bool,
	newHallReq messages.NewHallRequest) ([config.NUM_FLOORS][2]bool, bool) {
	// if button is invalid we return false
	if newHallReq.HallButton == elevator.ButtonCab {
		// fmt.Printf("Received a new hall request, but the button type was invalid\n")
		return globalHallRequests, false
	}
	// if the button is valid we update the global hall requests
	globalHallRequests[newHallReq.Floor][newHallReq.HallButton] = true
	return globalHallRequests, true
}
