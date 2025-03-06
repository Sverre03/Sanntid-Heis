package node

import (
	"elev/util/config"
	"fmt"
	"time"
)

func InactiveProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now Inactive\n", node.ID)

	for {
		select {

		case isDoorStuck := <-node.IsDoorStuckCh:
			if !isDoorStuck {
				return Disconnected
			}
		case <-time.After(config.NODE_DOOR_POLL_RATE):
			node.RequestDoorStateCh <- true

		// always make sure there are no receive channels in the node that are not present here
		case <-node.HallAssignmentsRx:
		case <-node.HallLightUpdateRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
		case <-node.ConnectionReqRx:
		case <-node.ConnectionReqAckRx:
		case <-node.ActiveElevStatesFromServerRx:
		case <-node.AllElevStatesFromServerRx:
		case <-node.TOLCFromServerRx:
		case <-node.ActiveNodeIDsFromServerRx:
		case <-node.NewHallReqRx:
		case <-node.ElevatorHallButtonEventRx:
		case <-node.ElevatorHRAStatesRx:
		case <-node.HallAssignmentCompleteRx:
		case <-node.HallAssignmentCompleteAckRx:
		}
	}
}
