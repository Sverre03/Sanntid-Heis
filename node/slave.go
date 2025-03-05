package node

import (
	"context"
	"elev/Network/network/messages"
	"elev/elevator"
	"elev/util/config"
	"fmt"
)

func SlaveProgram(node *NodeData) {
	fmt.Printf("Node %d is now a Slave\n", node.ID)
	lastHallAssignmentMessageID := uint64(0)

	for {
		select {
		case isDoorStuck := <-node.IsDoorStuckCh:
			if isDoorStuck {
				if err := node.NodeState.Event(context.Background(), "inactivate"); err != nil {
					fmt.Println("Error:", err)
				}
			}
		case newHA := <-node.HallAssignmentsRx:
			if newHA.NodeID != node.ID {
				break
			}
			node.AckTx <- messages.Ack{MessageID: newHA.MessageID, NodeID: node.ID}

			if lastHallAssignmentMessageID != newHA.MessageID {
				for i := 0; i < config.NUM_FLOORS; i++ {
					if newHA.HallAssignment[i][elevator.BT_HallUp] {
						node.ElevatorHallButtonEventTx <- elevator.ButtonEvent{Floor: i, Button: elevator.BT_HallUp}
					}
					if newHA.HallAssignment[i][elevator.BT_HallDown] {
						node.ElevatorHallButtonEventTx <- elevator.ButtonEvent{Floor: i, Button: elevator.BT_HallDown}
					}
				}
			}
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
