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
			// check whether the door is not stuck
			if !elevMsg.DoorIsStuck && elevMsg.EventType == singleelevator.DoorStuckEvent {
				nextNodeState = Disconnected
				break ForLoop
			}

		case <-node.HallAssignmentsRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
		case <-node.ConnectionReqRx:
		case <-node.NodeElevStateUpdate:
		case <-node.NewHallReqRx:
		case <-node.HallAssignmentCompleteRx:
		case <-node.NetworkEventRx:
		case <-node.MyElevStatesRx:

		}
	}
	return nextNodeState
}
