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
			fmt.Printf("Inactive received elevator event: %v, nodeID: %d\n", elevMsg, node.ID)
			// check whether the door is not stuck
			if !elevMsg.DoorIsStuck && node.ID == elevMsg.SourceNodeID && elevMsg.EventType == singleelevator.DoorStuckEvent {
				nextNodeState = Disconnected
				break ForLoop
			}

		case <-node.HallAssignmentsRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
		case <-node.ConnectionReqRx:
		case <-node.NodeElevStateUpdate:
		case <-node.NetworkEventRx:
		case <-node.MyElevStatesRx:

		}
	}
	return nextNodeState
}
