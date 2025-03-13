package node

import (
	"elev/Network/messages"
	"elev/elevator"
	"fmt"
)

func SlaveProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now a Slave\n", node.ID)
	lastHallAssignmentMessageID := uint64(0)

	var nextNodeState nodestate

	node.commandToServerTx <- "startConnectionTimeoutDetection"

ForLoop:
	for {
		select {
		case elevMsg := <-node.FromElevator:
			switch elevMsg.Type {
			case messages.MsgDoorStuck:
				if elevMsg.IsDoorStuck {
					nextNodeState = Inactive
					break ForLoop
				}

			case messages.MsgHallButtonEvent:
				node.NewHallReqTx <- messages.NewHallRequest{
					Floor:      elevMsg.ButtonEvent.Floor,
					HallButton: elevMsg.ButtonEvent.Button,
				}

			case messages.MsgHallAssignmentComplete:
				// Forward completed hall assignments
				if elevMsg.ButtonEvent.Button != elevator.ButtonCab {
					hallAssignmentCompleteMsg := messages.HallAssignmentComplete{
						Floor:      elevMsg.ButtonEvent.Floor,
						HallButton: elevMsg.ButtonEvent.Button,
						MessageID:  uint64(0), // Placeholder, Generate message ID as needed
					}

					node.HallAssignmentCompleteTx <- hallAssignmentCompleteMsg
					fmt.Printf("Node %d sent hall assignment complete message\n", node.ID)

					node.GlobalHallRequests[elevMsg.ButtonEvent.Floor][elevMsg.ButtonEvent.Button] = false
					node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
				}

			case messages.MsgElevatorState:
				// Forward elevator state to network
				node.ElevStatesTx <- messages.NodeElevState{
					NodeID:    node.ID,
					ElevState: elevMsg.ElevState,
				}
			}

		case timeout := <-node.ConnectionLossEventRx:
			if timeout {
				nextNodeState = Disconnected
				break ForLoop
			}

		case newHA := <-node.HallAssignmentsRx:
			if newHA.NodeID != node.ID {
				break
			}

			node.AckTx <- messages.Ack{MessageID: newHA.MessageID, NodeID: node.ID}

			if lastHallAssignmentMessageID != newHA.MessageID {
				node.ToElevator <- messages.NodeToElevatorMsg{
					Type:            messages.MsgHallAssignment,
					HallAssignments: newHA.HallAssignment,
				}
				lastHallAssignmentMessageID = newHA.MessageID
			}

		case lightUpdate := <-node.HallLightUpdateRx:
			// set the lights
			fmt.Println(lightUpdate)

		case hallReqFromMaster := <-node.GlobalHallRequestRx:
			node.GlobalHallRequests = hallReqFromMaster.HallRequests

		case <-node.ActiveElevStatesFromServerRx:
		case <-node.AllElevStatesFromServerRx:
		case <-node.NewHallReqRx:
		case <-node.TOLCFromServerRx:
		case <-node.ConnectionReqRx:
		case <-node.ConnectionReqAckRx:
		case <-node.CabRequestInfoRx:
		case <-node.ActiveNodeIDsFromServerRx:
		case <-node.HallAssignmentCompleteAckRx:
		}

	}
	return nextNodeState
}
