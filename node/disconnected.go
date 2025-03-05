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
	timeOfLastContact := time.Time{}                        // placeholder for getting from server
	msgID, _ := comm.GenerateMessageID(comm.CONNECTION_REQ) // placeholder for using "getmessageid function"

	myConnReq := messages.ConnectionReq{TOLC: timeOfLastContact, NodeID: node.ID, MessageID: msgID}
	incomingConnRequests := make(map[int]messages.ConnectionReq)

	// ID of the node we currently are trying to connect with
	currentFriendID := 0

	var lastReceivedAck *messages.Ack

	for {
		select {
		case <-node.GlobalHallRequestRx:
			// here, we must check if the master knows anything about us
			// this message transaction should be defined better than it is now, who sends what?
			if err := node.NodeState.Event(context.Background(), "connect"); err != nil {
				fmt.Println("Error:", err)
			} else {
				return
			}

		case incomingConnReq := <-node.ConnectionReqRx:
			if node.ID != incomingConnReq.NodeID {
				incomingConnRequests[incomingConnReq.NodeID] = incomingConnReq
				if currentFriendID == 0 || currentFriendID > incomingConnReq.NodeID {
					// this is the node with the lowest ID, I want to start a relationship with him
					currentFriendID = incomingConnReq.NodeID
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

			// timeout should be a const variable
		case <-time.After(time.Millisecond * 500):
			// start sending a conn request :)
			// isConnRequestActive = true
			node.ConnectionReqTx <- myConnReq
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
		} else if myID < otherID {
			return false
		}
	}
}
