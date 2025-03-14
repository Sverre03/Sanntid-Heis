package node

import (
	"elev/Network/messages"
	"elev/elevator"
	"elev/singleelevator"
	"elev/util/config"
	"fmt"
	"time"
)

func SlaveProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now a Slave\n", node.ID)
	lastHallAssignmentMessageID := uint64(0)

	var nextNodeState nodestate

	node.commandToServerTx <- "startConnectionTimeoutDetection"

ForLoop:
	for {
	Select:
		select {
		case elevMsg := <-node.ElevatorEventRx:

			switch elevMsg.EventType {
			case singleelevator.DoorStuckEvent:
				if elevMsg.IsDoorStuck {
					nextNodeState = Inactive
					break ForLoop
				}

			case singleelevator.HallButtonEvent:
				node.NewHallReqTx <- messages.NewHallRequest{
					Floor:      elevMsg.ButtonEvent.Floor,
					HallButton: elevMsg.ButtonEvent.Button,
				}

			case singleelevator.LocalHallAssignmentCompleteEvent:

				// Forward completed hall assignments
				if elevMsg.ButtonEvent.Button != elevator.ButtonCab {

					node.HallAssignmentCompleteTx <- messages.HallAssignmentComplete{
						Floor:      elevMsg.ButtonEvent.Floor,
						HallButton: elevMsg.ButtonEvent.Button,
						MessageID:  uint64(0),
					}
					fmt.Printf("Node %d sent hall assignment complete message\n", node.ID)
				}

			}

		case myElevStates := <-node.MyElevStatesRx:
			// Transmit elevator states to network
			node.ElevStatesTx <- messages.NodeElevState{
				NodeID:    node.ID,
				ElevState: myElevStates,
			}

		case timeout := <-node.ConnectionLossEventRx:
			if timeout {
				nextNodeState = Disconnected
				break ForLoop
			}

		case newHA := <-node.HallAssignmentsRx:
			if newHA.NodeID != node.ID ||
				!canAcceptHallAssignments(newHA.HallAssignment, node.GlobalHallRequests) {
				break Select
			}

			// the hall assignments are for me, so I can ack them
			node.AckTx <- messages.Ack{MessageID: newHA.MessageID, NodeID: node.ID}

			// lets check if I have already received this message, if not its update time!
			if lastHallAssignmentMessageID != newHA.MessageID {
				node.ElevAssignmentLightUpdateTx <- makeAssignmentAndLightMessage(newHA.HallAssignment, node.GlobalHallRequests)
				lastHallAssignmentMessageID = newHA.MessageID
			}

		case newGlobalHallReq := <-node.GlobalHallRequestRx:
			node.TOLC = time.Now()

			if hasChanged(newGlobalHallReq.HallRequests, node.GlobalHallRequests) {
				node.ElevAssignmentLightUpdateTx <- makeLightMessage(newGlobalHallReq)
				node.GlobalHallRequests = newGlobalHallReq.HallRequests
			}

		case <-node.ActiveElevStatesFromServerRx:
		case <-node.AllElevStatesFromServerRx:
		case <-node.NewHallReqRx:
		case <-node.ConnectionReqRx:
		case <-node.ConnectionReqAckRx:
		case <-node.CabRequestInfoRx:
		case <-node.ActiveNodeIDsFromServerRx:
		case <-node.HallAssignmentCompleteAckRx:
		}

	}
	return nextNodeState
}

func canAcceptHallAssignments(newHallAssignments, globalHallReq [config.NUM_FLOORS][2]bool) bool {
	for floor := 0; floor < config.NUM_FLOORS; floor++ {
		// check if my new assignment contains assignments that I am yet to be informed of from master
		if newHallAssignments[floor][elevator.ButtonHallDown] && !(globalHallReq[floor][elevator.ButtonHallDown]) {
			return false
		}
		if newHallAssignments[floor][elevator.ButtonHallUp] && !(globalHallReq[floor][elevator.ButtonHallUp]) {
			return false
		}
	}
	return true
}

func makeAssignmentAndLightMessage(hallAssignments [config.NUM_FLOORS][2]bool, globalHallReq [config.NUM_FLOORS][2]bool) singleelevator.LightAndHallAssignmentUpdate {
	var newMessage singleelevator.LightAndHallAssignmentUpdate
	newMessage.HallAssignmentAreNew = true
	newMessage.HallAssignments = hallAssignments
	newMessage.LightStates = globalHallReq
	return newMessage
}

func makeLightMessage(globalHallReq messages.GlobalHallRequest) singleelevator.LightAndHallAssignmentUpdate {
	var newMessage singleelevator.LightAndHallAssignmentUpdate
	newMessage.HallAssignmentAreNew = false
	newMessage.LightStates = globalHallReq.HallRequests
	return newMessage
}

func hasChanged(newGlobalHallReq, oldGlobalHallReq [config.NUM_FLOORS][2]bool) bool {
	for floor := 0; floor < config.NUM_FLOORS; floor++ {
		// check if the new is equal to the old or not
		if oldGlobalHallReq[floor][elevator.ButtonHallDown] != newGlobalHallReq[floor][elevator.ButtonHallDown] {
			return true
		}
		if oldGlobalHallReq[floor][elevator.ButtonHallUp] != newGlobalHallReq[floor][elevator.ButtonHallUp] {
			return true
		}
	}
	return false
}
