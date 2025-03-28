package node

import (
	"elev/singleelevator"
	"fmt"
)

func InactiveProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now Inactive\n", node.ID)
	var nextNodeState nodestate
ForLoop:
	for {
		select {

		case elevMsg := <-node.ElevatorEventRx:

			if !elevMsg.ElevIsDown && elevMsg.EventType == singleelevator.ElevStatusUpdateEvent { // If elevator is up , go to Disconnected state.
				nextNodeState = Disconnected
				break ForLoop
			}

		case <-node.HallAssignmentsRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
		case <-node.ConnectionReqRx:
		case <-node.ElevStateUpdatesFromServer:
		case <-node.NetworkEventRx:
		case <-node.MyElevStatesRx:
			// Read these to prevent blocking
		}
	}
	return nextNodeState
}
