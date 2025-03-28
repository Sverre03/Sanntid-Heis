package node

import (
	"elev/config"
	"elev/network/messages"
	"elev/singleelevator"
	"elev/util"
	"fmt"
	"time"
)

func DisconnectedProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now Disconnected\n", node.ID)

	// Set up a connectiong request message that are sent to other nodes to make contact
	myConnReq := messages.ConnectionReq{
		ContactCounterValue: node.ContactCounter,
		NodeID:              node.ID,
	}

	// Initializing a empty map with connection requests recieved from other nodes
	incomingConnRequests := make(map[int]messages.ConnectionReq)

	var nextNodeState nodestate

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
			} // ignore hall button events, we do not take new calls when disc

		case cabRequestInfo := <-node.CabRequestInfoRx:
			if cabRequestInfo.ReceiverNodeID == node.ID {
				// if the message was for us, we have established contact with a master and may now become slave
				// if we have never had any contact, we also restore cab orders from master
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
			// read these to prevent blocking
		}
	}
	return nextNodeState
}

// returns true if you have the most recent contact counter value, or you have an equivalent contact counter value to another node and the largest ID
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
