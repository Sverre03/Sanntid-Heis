package node

import (
	"elev/singleelevator"
	"elev/util/config"
	"fmt"
	"time"
)

func InactiveProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now Inactive\n", node.ID)
	var nextNodeState nodestate
ForLoop:
	for {
		select {

		case elevMsg := <-node.ElevatorEventRx:
			// check whether the door is not stuck
			if !elevMsg.IsDoorStuck && elevMsg.EventType == singleelevator.DoorStuckEvent {
				nextNodeState = Inactive
				break ForLoop
			}

		case <-time.After(config.NODE_DOOR_POLL_INTERVAL):
		case <-node.HallAssignmentsRx:
		case <-node.HallLightUpdateRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
		case <-node.ConnectionReqRx:
		case <-node.ConnectionReqAckRx:
		case <-node.ActiveElevStatesFromServerRx:
		case <-node.AllElevStatesFromServerRx:
		case <-node.ActiveNodeIDsFromServerRx:
		case <-node.NewHallReqRx:
		case <-node.HallAssignmentCompleteRx:
		case <-node.HallAssignmentCompleteAckRx:
		case <-node.ConnectionLossEventRx:
		}
	}
	return nextNodeState
}
