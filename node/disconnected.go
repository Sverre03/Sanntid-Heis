package node

import (
	"elev/Network/messagehandler"
	"elev/Network/messages"
	"elev/elevator"
	"elev/singleelevator"
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
	// ID of the node we currently are trying to connect with
	currentFriendID := 0

	var lastReceivedAck *messages.Ack

	// Set up heartbeat for connection requests
	connectionRequestTicker := time.NewTicker(500 * time.Millisecond)

	defer connectionRequestTicker.Stop()

ForLoop:
	for {
		select {
		case <-connectionRequestTicker.C: // Send connection request periodically
			node.ConnectionReqTx <- myConnReq

		case incomingConnReq := <-node.ConnectionReqRx:
			if node.ID != incomingConnReq.NodeID {
				fmt.Printf("Node %d received connection request from node %d\n",
					node.ID, incomingConnReq.NodeID)

				incomingConnRequests[incomingConnReq.NodeID] = incomingConnReq

				// Choose the node with lowest ID as potential connection
				if currentFriendID == 0 || currentFriendID >= incomingConnReq.NodeID {
					currentFriendID = incomingConnReq.NodeID
					// Send acknowledgement
					node.AckTx <- messages.Ack{
						MessageID: incomingConnReq.MessageID,
						NodeID:    node.ID,
					}
				}

			}

		case connReqAck := <-node.ConnectionReqAckRx:
			fmt.Printf("Node %d received connection request ack from node %d\n", node.ID, connReqAck.NodeID)
			if node.ID != connReqAck.NodeID && connReqAck.NodeID == currentFriendID {
				// All these decisions should be moved into a pure function, and the result returned
				// check who has the most recent data
				// here, we must ask on node.commandTx "getTOLC". Then, on return from node.TOLCRx compare
				lastReceivedAck = &connReqAck

				if lastReceivedAck != nil && node.ID != lastReceivedAck.NodeID && lastReceivedAck.NodeID == currentFriendID {

					if connReq, exists := incomingConnRequests[lastReceivedAck.NodeID]; exists {

						if ShouldBeMaster(node.ID, lastReceivedAck.NodeID, currentFriendID, node.TOLC, connReq.TOLC) {
							nextNodeState = Master
							fmt.Printf("Node %d is now a Master\n", node.ID)
						} else {
							fmt.Printf("Node %d is now a Slave\n", node.ID)
							nextNodeState = Slave
						}
						break ForLoop
					}
					lastReceivedAck = nil
				}
			}

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
			if node.ID == info.ReceiverNodeID && node.TOLC.IsZero() {
				// we have received info about us from the master, so we can become a slave

			}
			nextNodeState = Slave
			break ForLoop
		// check if you receive some useful info here
		// Prevent blocking of unused channels
		case <-node.HallAssignmentsRx:
		case <-node.HallLightUpdateRx:
		case <-node.AllElevStatesFromServerRx:
		case <-node.ActiveNodeIDsFromServerRx:
		case <-node.NewHallReqRx:
		case <-node.HallAssignmentCompleteRx:
		case <-node.HallAssignmentCompleteAckRx:
		case <-node.ActiveElevStatesFromServerRx:
		case <-node.ConnectionLossEventRx:
		case <-node.GlobalHallRequestRx:
		}
	}
	return nextNodeState
}

func ShouldBeMaster(myID int, otherID int, _currentFriendID int, TOLC time.Time, otherTOLC time.Time) bool {
	// Compare TOLC values to determine who becomes master
	if TOLC.Before(otherTOLC) { // We have the more recent data --> We should be master
		return true
	} else if TOLC.After(otherTOLC) { // We dont have more recent data --> We should be slave
		return false
	} else { // TOLC values are equal --> Compare node IDs
		if myID > otherID {
			return true
		} else {
			return false
		}
	}
}
