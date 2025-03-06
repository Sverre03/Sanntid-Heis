package node

import (
	"elev/Network/messages"
	"fmt"
)

func SlaveProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now a Slave\n", node.ID)
	lastHallAssignmentMessageID := uint64(0)

	for {
		select {
		case isDoorStuck := <-node.IsDoorStuckCh:
			if isDoorStuck {
				return Inactive
			}

		case timeout := <-node.ConnectionTimeoutEventRx:
			if timeout {
				return Disconnected
			}

		case newHA := <-node.HallAssignmentsRx:
			if newHA.NodeID != node.ID {
				break
			}

			node.AckTx <- messages.Ack{MessageID: newHA.MessageID, NodeID: node.ID}

			if lastHallAssignmentMessageID != newHA.MessageID {
				node.ElevatorHallButtonAssignmentTx <- newHA.HallAssignment
			}
		case lightUpdate := <-node.HallLightUpdateRx:
			// set the lights
			fmt.Println(lightUpdate)

		case hallReqFromMaster := <-node.GlobalHallRequestRx:
			node.GlobalHallRequests = hallReqFromMaster.HallRequests

		case btnEvent := <-node.ElevatorHallButtonEventRx:
			node.NewHallReqTx <- messages.NewHallRequest{Floor: btnEvent.Floor, HallButton: btnEvent.Button}

		case currentElevStates := <-node.MyElevatorStatesRx:
			node.ElevStatesTx <- messages.NodeElevState{NodeID: node.ID, ElevState: currentElevStates}
		case <-node.ActiveElevStatesFromServerRx:
		case <-node.AllElevStatesFromServerRx:
		case <-node.NewHallReqRx:
		case <-node.TOLCFromServerRx:
		case <-node.ConnectionReqRx:
		case <-node.ConnectionReqAckRx:
		case <-node.MyElevatorStatesRx:
		case <-node.CabRequestInfoRx:
		case <-node.ActiveNodeIDsFromServerRx:
		case <-node.HallAssignmentCompleteRx:
		case <-node.HallAssignmentCompleteAckRx:
		}

	}
}
