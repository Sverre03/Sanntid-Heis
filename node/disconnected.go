package node

import (
	"elev/Network/messagehandler"
	"elev/Network/messages"
	"elev/elevator"
	"elev/singleelevator"
	"elev/util/config"
	"fmt"
	"time"
)

func DisconnectedProgram(node *NodeData) nodestate {
	// note: this function could use a rewrite
	fmt.Printf("Node %d is now Disconnected\n", node.ID)

	connectionReqMsgID, _ := messagehandler.GenerateMessageID(messagehandler.CONNECTION_REQ)

	myConnReq := messages.ConnectionReq{
		TOLC:      node.TOLC,
		NodeID:    node.ID,
		MessageID: connectionReqMsgID,
	}
	incomingConnRequests := make(map[int]messages.ConnectionReq)
	var nextNodeState nodestate

	// Set up heartbeat for connection requests
	connectionRequestTicker := time.NewTicker(500 * time.Millisecond)
	decisionTimer := time.NewTimer(config.DISCONNECTED_DECISION_INTERVAL)
	defer connectionRequestTicker.Stop()

ForLoop:
	for {
		select {
		case <-connectionRequestTicker.C: // Send connection request periodically
			node.ConnectionReqTx <- myConnReq

		case incomingConnReq := <-node.ConnectionReqRx:
			if node.ID != incomingConnReq.NodeID {
				// fmt.Printf("Node %d received connection request from node %d\n",node.ID, incomingConnReq.NodeID)

				incomingConnRequests[incomingConnReq.NodeID] = incomingConnReq
			}

		case <-decisionTimer.C:
			if len(incomingConnRequests) != 0 {
				if ShouldBeMaster(node.ID, node.TOLC, incomingConnRequests) {
					nextNodeState = Master
				} else {
					nextNodeState = Slave
				}
				break ForLoop
			}
			fmt.Printf("No contact made so far \n")
			decisionTimer.Reset(config.DISCONNECTED_DECISION_INTERVAL)

		case elevMsg := <-node.ElevatorEventRx:
			switch elevMsg.EventType {

			case singleelevator.DoorStuckEvent:
				if elevMsg.DoorIsStuck {
					nextNodeState = Inactive
					break ForLoop
				}

			case singleelevator.HallButtonEvent:
				// ignore hall button presses

			case singleelevator.LocalHallAssignmentCompleteEvent:
				// update the global hall requests, it is safe as we are now disconnected
				if elevMsg.ButtonEvent.Button != elevator.ButtonCab {
					node.GlobalHallRequests[elevMsg.ButtonEvent.Floor][elevMsg.ButtonEvent.Button] = false
				}
			}

		case info := <-node.CabRequestInfoRx: // Check if the master has any info about us
			fmt.Println("I found a master, time to be a slave")
			if node.ID == info.ReceiverNodeID && node.TOLC.IsZero() {
				// we have received info about us from the master, so we can become a slave
				node.ElevLightAndAssignmentUpdateTx <- makeCabOrderMessage(info.CabRequest)
			}
			nextNodeState = Slave
			break ForLoop
		case <-node.HallAssignmentsRx:
		case <-node.NodeElevStateUpdate:
		case <-node.NewHallReqRx:
		case <-node.HallAssignmentCompleteRx:
		case <-node.NetworkEventRx:
		case <-node.GlobalHallRequestRx:
		case <-node.MyElevStatesRx:

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
