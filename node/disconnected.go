package node

import (
	"elev/config"
	"elev/network/messages"
	"elev/singleelevator"
	"elev/util"
	"time"
)

// DisconnectedProgram runs when the node is searching for other nodes to connect to.
// It broadcasts connection requests and determines if it should become master based on incoming connection requests.
// In Disconnected state, the node operates as a standalone elevator.
// It does not take new hall calls, but keeps track of its already existing hall assignments.
func DisconnectedProgram(node *NodeData) NodeState {

	// Set up a connectiong request message that are sent to other nodes to make contact
	myConnReq := messages.ConnectionReq{
		ContactCounterValue: node.ContactCounter,
		NodeID:              node.ID,
	}

	// Initializing a empty map with connection requests recieved from other nodes
	incomingConnRequests := make(map[int]messages.ConnectionReq)

	var nextNodeState NodeState

	// Set up heartbeat for connection requests
	connRequestTicker := time.NewTicker(config.CONNECTION_REQ_INTERVAL)
	decisionTimer := time.NewTimer(config.STATE_TRANSITION_DECISION_INTERVAL)
	defer connRequestTicker.Stop()

	// Doing my own hall assignments
	node.ElevLightAndAssignmentUpdateTx <- makeHallAssignmentAndLightMessage(node.GlobalHallRequests, node.GlobalHallRequests, -1)

ForLoop:
	for {
		select {
		case <-connRequestTicker.C: // Send connection request periodically
			node.ConnectionReqTx <- myConnReq

		case incomingConnReq := <-node.ConnectionReqRx:
			if node.ID != incomingConnReq.NodeID { // Check if message is from other node
				incomingConnRequests[incomingConnReq.NodeID] = incomingConnReq
			}

		case <-decisionTimer.C: // Waits for the decision timer to run out
			if !util.MapIsEmpty(incomingConnRequests) { // Checks if  there are incoming connection requests
				if ShouldBeMaster(node.ID, node.ContactCounter, incomingConnRequests) { //  Check if this node should become the master
					nextNodeState = Master
					break ForLoop
				}
			}
			// Otherwise, reset the decision timer for the next evaluation cycle
			decisionTimer.Reset(config.STATE_TRANSITION_DECISION_INTERVAL)

		case elevMsg := <-node.ElevatorEventRx:
			switch elevMsg.EventType {
			// Handle elevator status update; go Inactive if elevator is down.
			case singleelevator.ElevStatusUpdateEvent:
				if elevMsg.ElevIsDown {
					nextNodeState = Inactive
					break ForLoop
				}
			} // Ignore hall button events, we do not take new calls when disc

		case cabRequestInfo := <-node.CabRequestInfoRx:
			if cabRequestInfo.ReceiverNodeID == node.ID {
				// If the message was for us, we have established contact with a master and may now become slave
				// If we have never had any contact, we also restore cab orders from master
				if node.ContactCounter == 0 {
					node.ElevLightAndAssignmentUpdateTx <- makeCabAssignmentMessage(cabRequestInfo.CabRequest)
				}
				nextNodeState = Slave
				break ForLoop
			}

		case elevStates := <-node.MyElevStatesRx:

			if util.HallAssignmentIsRemoved(node.GlobalHallRequests, elevStates.MyHallAssignments) {
				node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(elevStates.MyHallAssignments)
			}
			node.GlobalHallRequests = elevStates.MyHallAssignments

		case <-node.HallAssignmentsRx:
		case <-node.ElevStateUpdatesFromServer:
		case <-node.NetworkEventRx:
		case <-node.GlobalHallRequestRx:
		case <-node.NewHallReqRx:
			// Read these to prevent blocking
		}
	}
	return nextNodeState
}

// Returns true if you have the most recent contact counter value,
// or if you have an equivalent contact counter value to another node and the largest ID
func ShouldBeMaster(myID int, contactCounter uint64, connectionRequests map[int]messages.ConnectionReq) bool {

	for _, connReq := range connectionRequests {
		if util.MyCounterIsSmaller(contactCounter, connReq.ContactCounterValue) {
			return false
		}
		if contactCounter == connReq.ContactCounterValue {
			if myID < connReq.NodeID {
				return false
			}
		}
	}

	return true
}

func makeCabAssignmentMessage(cabRequests [config.NUM_FLOORS]bool) singleelevator.LightAndAssignmentUpdate {
	return singleelevator.LightAndAssignmentUpdate{
		CabAssignments:  cabRequests,
		LightStates:     [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool{},
		OrderType:       singleelevator.CabAssignment,
		HallAssignments: [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool{},
	}
}
