package node

import (
	"elev/Network/messages"
	"elev/config"
	"elev/singleelevator"
	"fmt"
	"time"
)

func DisconnectedProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now Disconnected\n", node.ID)

	myConnReq := messages.ConnectionReq{
		TOLC:      node.TOLC,
		NodeID:    node.ID,
	}
	incomingConnRequests := make(map[int]messages.ConnectionReq)
	
	var nextNodeState nodestate

	// Set up heartbeat for connection requests
	connRequestTicker := time.NewTicker(config.CONNECTION_REQ_INTERVAL)
	decisionTimer := time.NewTimer(config.DISCONNECTED_DECISION_INTERVAL)
	defer connRequestTicker.Stop()

	// Doing my own hall assignments
	node.ElevLightAndAssignmentUpdateTx <- makeHallAssignmentAndLightMessage(node.GlobalHallRequests, node.GlobalHallRequests)

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
			if !isMapEmtpy(incomingConnRequests) {
				if ShouldBeMaster(node.ID, node.TOLC, incomingConnRequests) {
					nextNodeState = Master
					break ForLoop
				}
			} else {
				fmt.Printf("No contact made so far\n")
			}
			decisionTimer.Reset(config.DISCONNECTED_DECISION_INTERVAL)

		case elevMsg := <-node.ElevatorEventRx:
			if isDoorStuck(elevMsg) {
				nextNodeState = Inactive
				break ForLoop
			}

		case cabRequestInfo := <-node.CabRequestInfoRx: // Check if the master has any info about us
			fmt.Println("Master found -> go to Slave")
			if cabRequestInfoForMe(cabRequestInfo, node) {
				// we have received info about us from the master, so we can become a slave
				node.ElevLightAndAssignmentUpdateTx <- makeCabOrderMessage(cabRequestInfo.CabRequest)
			}
			nextNodeState = Slave
			break ForLoop
		case <-node.HallAssignmentsRx:
		case <-node.NodeElevStateUpdate:
		case <-node.NetworkEventRx:
		case <-node.GlobalHallRequestRx:
		case <-node.MyElevStatesRx:
		case <-node.NewHallReqRx:

		}
	}
	return nextNodeState
}

func ShouldBeMaster(myID int, tolc time.Time, connectionRequests map[int]messages.ConnectionReq) bool {
	if tolc.IsZero() {
		for id, connReq := range connectionRequests {
			if id > myID || !connReq.TOLC.IsZero() {
				return false
			}
		}
		return true
	}

	for _, connReq := range connectionRequests {
		if tolc.Before(connReq.TOLC) {
			return false
		}
	}
	return true
}

func makeCabOrderMessage(cabRequests [config.NUM_FLOORS]bool) singleelevator.LightAndAssignmentUpdate {
	return singleelevator.LightAndAssignmentUpdate{
		CabAssignments:  cabRequests,
		LightStates:     [config.NUM_FLOORS][2]bool{},
		OrderType:       singleelevator.CabOrder,
		HallAssignments: [config.NUM_FLOORS][2]bool{},
	}
}

func isMapEmtpy(m map[int]messages.ConnectionReq) bool {
	return len(m) == 0
}	

func isDoorStuck(elevMsg singleelevator.ElevatorEvent) bool {
	return elevMsg.DoorIsStuck && elevMsg.EventType == singleelevator.DoorStuckEvent
}

func cabRequestInfoForMe(cabRequestInfo messages.CabRequestInfo, node *NodeData) bool {
	return node.ID == cabRequestInfo.ReceiverNodeID && node.TOLC.IsZero()
}