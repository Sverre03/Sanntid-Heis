package node

import (
	"elev/Network/messagehandler"
	"elev/Network/messages"
	"fmt"
	"time"
)

func DisconnectedProgram(node *NodeData) nodestate {
	// note: this function could use a rewrite
	fmt.Printf("Node %d is now Disconnected\n", node.ID)

	timeOfLastContact := time.Time{}
	msgID, _ := messagehandler.GenerateMessageID(messagehandler.CONNECTION_REQ)

	node.commandToServerTx <- "getTOLC"
	timeOfLastContact = <-node.TOLCFromServerRx

	myConnReq := messages.ConnectionReq{
		TOLC:      timeOfLastContact,
		NodeID:    node.ID,
		MessageID: msgID,
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
			if node.ID != connReqAck.NodeID && connReqAck.NodeID == currentFriendID {
				// All these decisions should be moved into a pure function, and the result returned
				// check who has the most recent data
				// here, we must ask on node.commandTx "getTOLC". Then, on return from node.TOLCRx compare
				lastReceivedAck = &connReqAck

				if lastReceivedAck != nil && node.ID != lastReceivedAck.NodeID && lastReceivedAck.NodeID == currentFriendID {

					if connReq, exists := incomingConnRequests[lastReceivedAck.NodeID]; exists {

						if ShouldBeMaster(node.ID, lastReceivedAck.NodeID, currentFriendID, timeOfLastContact, connReq.TOLC) {
							nextNodeState = Master
						} else {
							nextNodeState = Slave
						}
						break ForLoop
					}
					lastReceivedAck = nil
				}
			}

		case elevMsg := <-node.FromElevator:
			switch elevMsg.Type {
			case messages.MsgDoorStuck:
				if elevMsg.IsDoorStuck {
					nextNodeState = Inactive
					break ForLoop
				}
			// Else do nothing
			case messages.MsgHallButtonEvent:
				// do smth
			case messages.MsgHallAssignmentComplete:
				// do smth
			case messages.MsgElevatorState:
				// do smth
			}

		case <-node.GlobalHallRequestRx:
			// here, we must check if the master knows anything about us, before we become a slave
			if timeOfLastContact.IsZero() {
				// do smth
			}
			nextNodeState = Slave
			break ForLoop

		case info := <-node.CabRequestInfoRx:
			if node.ID == info.ReceiverNodeID {
				// do smth with it
				nextNodeState = Slave
				break ForLoop
			}
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
