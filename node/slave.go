package node

import (
	"elev/Network/messagehandler"
	"elev/Network/messages"
	"elev/elevator"
	"elev/singleelevator"
	"elev/util/config"
	"fmt"
	"time"
)

func SlaveProgram(node *NodeData) nodestate {
	// fmt.Printf("Node %d is now a Slave\n", node.ID)
	lastHallAssignmentMessageID := uint64(0)

	var nextNodeState nodestate

	// start the transmitters
	node.HallAssignmentCompleteTransmitEnableTx <- true

	node.commandToServerTx <- "startConnectionTimeoutDetection"

ForLoop:
	for {
	Select:
		select {
		case elevMsg := <-node.ElevatorEventRx:

			switch elevMsg.EventType {
			case singleelevator.DoorStuckEvent:
				if elevMsg.DoorIsStuck {
					nextNodeState = Inactive
					break ForLoop
				}

			case singleelevator.HallButtonEvent:
				node.NewHallReqTx <- messages.NewHallRequest{
					Floor:      elevMsg.ButtonEvent.Floor,
					HallButton: elevMsg.ButtonEvent.Button,
				}

			case singleelevator.LocalHallAssignmentCompleteEvent:
				fmt.Println("LocalHallAssignmentCompleteEvent")
				// Forward completed hall assignments
				if elevMsg.ButtonEvent.Button != elevator.ButtonCab {

					node.HallAssignmentCompleteTx <- messages.HallAssignmentComplete{
						Floor:      elevMsg.ButtonEvent.Floor,
						HallButton: elevMsg.ButtonEvent.Button,
						MessageID:  uint64(0),
					}
					// fmt.Printf("Node %d sent hall assignment complete message\n", node.ID)
				}

			}

		case myElevStates := <-node.MyElevStatesRx:
			// Transmit elevator states to network
			node.NodeElevStatesTx <- messages.NodeElevState{
				NodeID:    node.ID,
				ElevState: myElevStates,
			}

		case networkEvent := <-node.NetworkEventRx:
			if networkEvent == messagehandler.NodeHasLostConnection {
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
				node.ElevLightAndAssignmentUpdateTx <- makeHallAssignmentAndLightMessage(newHA.HallAssignment, node.GlobalHallRequests)
				lastHallAssignmentMessageID = newHA.MessageID
			}

		case newGlobalHallReq := <-node.GlobalHallRequestRx:
			node.TOLC = time.Now()
			// fmt.Println(newGlobalHallReq)
			if hasChanged(newGlobalHallReq.HallRequests, node.GlobalHallRequests) {
				node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(newGlobalHallReq.HallRequests)
				node.GlobalHallRequests = newGlobalHallReq.HallRequests
				// fmt.Printf("New global hall request: %v\n", node.GlobalHallRequests)
			}

		case <-node.NodeElevStateUpdate:
		case <-node.NewHallReqRx:
		case <-node.ConnectionReqRx:
		case <-node.ConnectionReqAckRx:
		case <-node.CabRequestInfoRx:
		}

	}

	// stop transmitters
	node.HallAssignmentCompleteTransmitEnableTx <- false
	if nextNodeState == Disconnected {
		fmt.Println("Exiting slave to disconnected")
	} else {
		fmt.Println("Exiting slave to inactive")
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

func makeHallAssignmentAndLightMessage(hallAssignments [config.NUM_FLOORS][2]bool, globalHallReq [config.NUM_FLOORS][2]bool) singleelevator.LightAndAssignmentUpdate {
	var newMessage singleelevator.LightAndAssignmentUpdate
	newMessage.HallAssignments = hallAssignments
	newMessage.LightStates = globalHallReq
	newMessage.OrderType = singleelevator.HallOrder
	return newMessage
}

func makeLightMessage(hallReq [config.NUM_FLOORS][2]bool) singleelevator.LightAndAssignmentUpdate {
	var newMessage singleelevator.LightAndAssignmentUpdate
	newMessage.LightStates = hallReq
	newMessage.OrderType = singleelevator.LightUpdate
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
