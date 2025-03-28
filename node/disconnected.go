package node

import (
	"elev/config"
	"elev/network/messages"
	"elev/singleelevator"
	"elev/util"
	"time"
)

// DisconnectedProgram searches for other nodes to connect with in the network.
// It operates as a standalone elevator and does not take new hall calls, but keeps track of its existing hall assignments.
// It broadcasts connection requests and determines if it should become master based on incoming connection requests.
func DisconnectedProgram(node *NodeData) NodeState {

	// Create connection request message with node's current state, to be sent to other nodes to make contact
	myConnReq := messages.ConnectionReq{
		ContactCounterValue: node.ContactCounter,
		NodeID:              node.ID,
	}

	// Track connection requests from other nodes
	incomingConnRequests := make(map[int]messages.ConnectionReq)
	var nextNodeState NodeState

	// Set up heartbeat for connection requests
	connRequestTicker := time.NewTicker(config.CONNECTION_REQ_INTERVAL)
	decisionTimer := time.NewTimer(config.STATE_TRANSITION_DECISION_INTERVAL)
	defer connRequestTicker.Stop()

	// Send initial hall assignments to elevator
	node.ElevLightAndAssignmentUpdateTx <- makeHallAssignmentAndLightMessage(node.GlobalHallRequests, node.GlobalHallRequests, -1)

MainLoop:
	for {
		select {
		case <-connRequestTicker.C: // Broadcast connection request periodically
			node.ConnectionReqTx <- myConnReq

		case incomingConnReq := <-node.ConnectionReqRx:
			if node.ID != incomingConnReq.NodeID { // Record incoming connection requests from other nodes
				incomingConnRequests[incomingConnReq.NodeID] = incomingConnReq
			}

		case <-decisionTimer.C: // Evaluate if this node should become master
			if !util.MapIsEmpty(incomingConnRequests) { // If we have received connection requests
				if ShouldBeMaster(node.ID, node.ContactCounter, incomingConnRequests) {
					nextNodeState = Master // Become master if we have the highest priority
					break MainLoop
				}
			}
			// Otherwise, reset the decision timer for the next evaluation cycle
			decisionTimer.Reset(config.STATE_TRANSITION_DECISION_INTERVAL)

		case elevMsg := <-node.ElevatorEventRx:
			// Check if elevator is no longer operational
			if elevMsg.ElevIsDown && elevMsg.EventType == singleelevator.ElevStatusUpdateEvent {
				nextNodeState = Inactive
				break MainLoop
			} // Ignore hall button events, we do not take new hall calls when Disconnected

		case cabRequestInfo := <-node.CabRequestInfoRx:
			// If the message was for us, we have established contact with a Master and may now become Slave
			if cabRequestInfo.ReceiverNodeID == node.ID {
				if node.ContactCounter == 0 {
					// First contact: restore cab assignments from Master
					node.ElevLightAndAssignmentUpdateTx <- makeCabAssignmentMessage(cabRequestInfo.CabRequest)
				}
				nextNodeState = Slave
				break MainLoop
			}

		case elevStates := <-node.MyElevStatesRx:
			if util.HallAssignmentIsRemoved(node.GlobalHallRequests, elevStates.MyHallAssignments) {
				node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(elevStates.MyHallAssignments)
			}
			node.GlobalHallRequests = elevStates.MyHallAssignments

		// Drain channels to prevent blocking
		case <-node.HallAssignmentsRx:
		case <-node.ElevStateUpdatesFromServer:
		case <-node.NetworkEventRx:
		case <-node.GlobalHallRequestRx:
		case <-node.NewHallReqRx:
		}
	}
	return nextNodeState
}

// ShouldBeMaster determines if this node should assume the master role
// based on having the highest contact counter or the highest ID when counters match.
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

// makeCabAssignmentMessage creates a message to update cab assignments.
func makeCabAssignmentMessage(cabRequests [config.NUM_FLOORS]bool) singleelevator.LightAndAssignmentUpdate {
	return singleelevator.LightAndAssignmentUpdate{
		CabAssignments:  cabRequests,
		LightStates:     [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool{},
		OrderType:       singleelevator.CabAssignment,
		HallAssignments: [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool{},
	}
}
