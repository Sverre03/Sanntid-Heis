package node

import (
	"context"
	"elev/util/config"
	"fmt"
	"time"
)

func InactiveProgram(node *NodeData) {
	fmt.Printf("Node %d is now Inactive\n", node.ID)
	if err := node.NodeState.Event(context.Background(), "initialize"); err != nil {
		fmt.Println("Error:", err)
	}

	for {
		select {

		case isDoorStuck := <-node.IsDoorStuckCh:
			if !isDoorStuck {
				if err := node.NodeState.Event(context.Background(), "activate"); err != nil {
					fmt.Println("Error:", err)
				}
			}
		case <-time.After(config.NODE_DOOR_POLL_RATE):
			node.RequestDoorStateCh <- true

		case <-node.HallAssignmentsRx:
		case <-node.HallLightUpdateRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
		case <-node.ConnectionReqRx:
		case <-node.ConnectionReqAckRx:
		case <-node.ActiveElevStatesRx:
		case <-node.AllElevStatesRx:
		case <-node.TOLCRx:
		case <-node.ActiveNodeIDsRx:
		case <-node.NewHallReqRx:
		case <-node.ElevatorHallButtonEventRx:
		case <-node.ElevatorHRAStatesRx:
		case <-node.HallAssignmentCompleteRx:
		case <-node.HallAssignmentCompleteAckRx:

		}
	}
}
