package node

import (
	"elev/Network/messages"
	"elev/util/config"
	"fmt"
	"time"
)

func InactiveProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now Inactive\n", node.ID)

	for {
		select {

		case elevMsg := <-node.FromElevator:
			switch elevMsg.Type {
			case messages.MsgDoorStuck:
				if elevMsg.IsDoorStuck {
					return Disconnected
				}
			// Else do nothing
			case messages.MsgHallButtonEvent:
				// do smth
			case messages.MsgHallAssignmentComplete:
				// do smth
			case messages.MsgElevatorState:
				// do smth
			}

		case <-time.After(config.NODE_DOOR_POLL_INTERVAL):
			// node.RequestDoorStateCh <- true
			node.ToElevator <- messages.NodeToElevatorMsg{
				Type: messages.MsgRequestDoorState,
			}

		// always make sure there are no receive channels in the node that are not present here
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
}
