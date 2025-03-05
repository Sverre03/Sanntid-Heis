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
		case lightUpdate := <-node.HallLightUpdateRx:
			// set the lights
			fmt.Println(lightUpdate)

		case hallReqFromMaster := <-node.GlobalHallRequestRx:
			node.GlobalHallRequests = hallReqFromMaster.HallRequests

		case btnEvent := <-node.ElevatorHallButtonEventRx:
			node.NewHallReqTx <- messages.NewHallRequest{Floor: btnEvent.Floor, HallButton: btnEvent.Button}

		case <-node.ActiveElevStatesRx:
		case <-node.AllElevStatesRx:
		case <-node.NewHallReqRx:
		case <-node.TOLCRx:
		case <-node.ConnectionReqRx:
		case <-node.ConnectionReqAckRx:
		case <-node.ElevatorHRAStatesRx:
		case <-node.CabRequestInfoRx:
		case <-node.ActiveNodeIDsRx:
		case <-node.HallAssignmentCompleteRx:
		case <-node.HallAssignmentCompleteAckRx:
		}

	}
}
