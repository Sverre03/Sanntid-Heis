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

	myConnReq := messages.ConnectionReq{
		ContactCounterValue: node.ContactCounter,
		NodeID:              node.ID,
	}
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
			if node.ID != incomingConnReq.NodeID {
				incomingConnRequests[incomingConnReq.NodeID] = incomingConnReq
			}

		case <-decisionTimer.C:
			if !util.MapIsEmpty(incomingConnRequests) {
				if ShouldBeMaster(node.ID, node.ContactCounter, incomingConnRequests) {
					nextNodeState = Master
					break ForLoop
				}
			}
			decisionTimer.Reset(config.STATE_TRANSITION_DECISION_INTERVAL)

		case elevMsg := <-node.ElevatorEventRx:
			switch elevMsg.EventType {
			case singleelevator.ElevStatusUpdateEvent:
				if elevMsg.ElevIsDown {
					nextNodeState = Inactive
					break ForLoop
				}

			}

		case cabRequestInfo := <-node.CabRequestInfoRx: // Check if the master has any info about us
			if cabRequestInfo.ReceiverNodeID == node.ID {
				if node.ContactCounter == 0 {
					node.ElevLightAndAssignmentUpdateTx <- makeCabOrderMessage(cabRequestInfo.CabRequest)
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
		case <-node.NodeElevStateUpdate:
		case <-node.NetworkEventRx:
		case <-node.GlobalHallRequestRx:
		case <-node.NewHallReqRx:

		}
	}
	return nextNodeState
}

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

func makeCabOrderMessage(cabRequests [config.NUM_FLOORS]bool) singleelevator.LightAndAssignmentUpdate {
	return singleelevator.LightAndAssignmentUpdate{
		CabAssignments:  cabRequests,
		LightStates:     [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool{},
		OrderType:       singleelevator.CabOrder,
		HallAssignments: [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool{},
	}
}
