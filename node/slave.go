package node

import (
	"elev/Network/messagehandler"
	"elev/Network/messages"
	"elev/config"
	"elev/elevator"
	"elev/singleelevator"
	"fmt"
	"time"
)

func SlaveProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now Slave\n", node.ID)

	lastHallAssignmentMessageID := uint64(0)

	var nextNodeState nodestate

	masterConnectionTimeoutTimer := time.NewTimer(config.MASTER_CONNECTION_TIMEOUT)
	masterConnectionTimeoutTimer.Stop()

	// start the transmitters
	node.commandToServerTx <- "startConnectionTimeoutDetection"
	// set them lights

ForLoop:
	for {
	Select:
		select {
		case elevMsg := <-node.ElevatorEventRx:
			switch elevMsg.EventType {
			case singleelevator.DoorStuckEvent:
				if elevMsg.DoorIsStuck && node.ID == elevMsg.SourceNodeID {
					fmt.Printf("Slave %d received door stuck event with source node id %d\n", node.ID, elevMsg.SourceNodeID)
					nextNodeState = Inactive
					break ForLoop
				}

			case singleelevator.HallButtonEvent:

				node.NewHallReqTx <- messages.NewHallReq{
					NodeID: node.ID,
					HallReq: elevator.ButtonEvent{
						Floor:  elevMsg.ButtonEvent.Floor,
						Button: elevMsg.ButtonEvent.Button,
					},
				}
				debug := messages.NewHallReq{
					NodeID: node.ID,
					HallReq: elevator.ButtonEvent{
						Floor:  elevMsg.ButtonEvent.Floor,
						Button: elevMsg.ButtonEvent.Button,
					},
				}
				fmt.Printf("Node %d received hall button event: %v\n", node.ID, debug.HallReq)

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
			fmt.Printf("New hall assignments: %v \n", newHA.HallAssignment)

			// the hall assignments are for me, so I can ack them
			node.AckTx <- messages.Ack{MessageID: newHA.MessageID, NodeID: node.ID}

			// lets check if I have already received this message, if not its update time!
			if lastHallAssignmentMessageID != newHA.MessageID {
				fmt.Println("Sending my update to the elev!")
				node.ElevLightAndAssignmentUpdateTx <- makeHallAssignmentAndLightMessage(newHA.HallAssignment, node.GlobalHallRequests)
				lastHallAssignmentMessageID = newHA.MessageID
			}

		case newGlobalHallReq := <-node.GlobalHallRequestRx:
			node.TOLC = time.Now()
			if hasChanged(newGlobalHallReq.HallRequests, node.GlobalHallRequests) {
				node.GlobalHallRequests = newGlobalHallReq.HallRequests
				// fmt.Printf("New global hall request: %v\n", node.GlobalHallRequests)
				node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(newGlobalHallReq.HallRequests)
			}
			masterConnectionTimeoutTimer.Reset(config.MASTER_CONNECTION_TIMEOUT)

		case <-masterConnectionTimeoutTimer.C:
			fmt.Printf("Node %d timed out\n", node.ID)
			nextNodeState = Disconnected
			break ForLoop

		case <-node.NodeElevStateUpdate:
		case <-node.ConnectionReqRx:
		case <-node.CabRequestInfoRx:
		case <-node.NewHallReqRx:
		}

	}

	// stop transmitters
	if nextNodeState == Disconnected {
		fmt.Println("Exiting slave to disconnected")
	} else {
		fmt.Println("Exiting slave to inactive")
	}
	node.TOLC = time.Now()
	return nextNodeState
}

func canAcceptHallAssignments(newHallAssignments, globalHallReq [config.NUM_FLOORS][2]bool) bool {
	for floor := range config.NUM_FLOORS {
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
	for floor := range config.NUM_FLOORS {
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
