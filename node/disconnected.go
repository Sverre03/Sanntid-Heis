package node

import (
	"context"
	"elev/Network/comm"
	"elev/Network/network/messages"
	"fmt"
	"time"
)

func DisconnectedProgram(node *NodeData) {
	fmt.Printf("Node %d is now Disconnected\n", node.ID)
	timeOfLastContact := time.Time{} // placeholder for getting from server
	msgID, _ := comm.GenerateMessageID(comm.CONNECTION_REQ)

	myConnReq := messages.ConnectionReq{
		TOLC:      timeOfLastContact,
		NodeID:    node.ID,
		MessageID: msgID,
	}
	incomingConnRequests := make(map[int]messages.ConnectionReq)

	// ID of the node we currently are trying to connect with
	currentFriendID := 0

	var lastReceivedAck *messages.Ack

	// Set up heartbeat for connection requests
	connectionRequestTicker := time.NewTicker(500 * time.Millisecond)
	defer connectionRequestTicker.Stop()

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
				node.commandTx <- "getTOLC"
			}

		case TOLC := <-node.TOLCRx:
			if lastReceivedAck != nil && node.ID != lastReceivedAck.NodeID && lastReceivedAck.NodeID == currentFriendID {
				if connReq, exists := incomingConnRequests[lastReceivedAck.NodeID]; exists {
					shouldBeMaster := ShouldBeMaster(node.ID, lastReceivedAck.NodeID, currentFriendID, TOLC, connReq.TOLC)
					if shouldBeMaster {
						if err := node.NodeState.Event(context.Background(), "promote"); err != nil {
							fmt.Println("Error:", err)
						}
					} else {
						if err := node.NodeState.Event(context.Background(), "connect"); err != nil {
							fmt.Println("Error:", err)
						}
					}
				}
				lastReceivedAck = nil
			}

		case <-node.GlobalHallRequestRx:
			// here, we must check if the master knows anything about us
			// this message transaction should be defined better than it is now, who sends what?
			if err := node.NodeState.Event(context.Background(), "connect"); err != nil {
				fmt.Println("Error:", err)
			} else {
				return
			}

		case isDoorStuck := <-node.IsDoorStuckCh:
			if isDoorStuck {
				if err := node.NodeState.Event(context.Background(), "inactivate"); err != nil {
					fmt.Println("Error:", err)
				}
			}

		// Prevent blocking of unused channels
		case <-node.HallAssignmentsRx:
		case <-node.RequestDoorStateCh:
		case <-node.CabRequestInfoRx:
		case <-node.HallLightUpdateRx:
		case <-node.ElevatorHRAStatesRx:
		case <-node.AllElevStatesRx:
		case <-node.ActiveNodeIDsRx:
		case <-node.NewHallReqRx:
		case <-node.HallAssignmentCompleteRx:
		case <-node.HallAssignmentCompleteAckRx:
		case <-node.ElevatorHallButtonEventRx:
		case <-node.ActiveElevStatesRx:
		}
	}
}

func ShouldBeMaster(myID int, otherID int, _currentFriendID int, TOLC time.Time, otherTOLC time.Time) bool {
	// Compare TOLC values to determine who becomes master
	if TOLC.Before(otherTOLC) { // The other node has more recent data --> We should be master
		return true
	} else if TOLC.After(otherTOLC) { // We have more recent data --> We should be slave
		return false
	} else { // TOLC values are equal --> Compare node IDs
		if myID > otherID {
			return true
		} else {
			return false
		}
	}
}
