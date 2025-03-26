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
	select {
	case node.commandToServerTx <- "startConnectionTimeoutDetection":
		// Command sent successfully
	default:
		// Command not sent, channel is full
		fmt.Printf("Warning: Command channel is full, command %s not sent\n", "startConnectionTimeoutDetection")
	}

ForLoop:
	for {
		select {
		case elevMsg := <-node.ElevatorEventRx:
			switch elevMsg.EventType {
			case singleelevator.DoorStuckEvent:
				if doorIsStuck(elevMsg) {
					nextNodeState = Inactive
					break ForLoop
				}

			case singleelevator.HallButtonEvent:
				node.NewHallReqTx <- makeNewHallReq(node.ID, elevMsg)
			}

		case myElevStates := <-node.MyElevStatesRx:
			// Transmit elevator states to network
			node.NodeElevStatesTx <- messages.NodeElevState{
				NodeID:    node.ID,
				ElevState: myElevStates,
			}

		case networkEvent := <-node.NetworkEventRx:
			if iHaveLostConenction(networkEvent) {
				nextNodeState = Disconnected
				break ForLoop
			}

		case newHA := <-node.HallAssignmentsRx:
			if canAcceptHallAssignments(newHA, node.GlobalHallRequests, node.ID) {
				// the hall assignments are for me, so I can ack them
				node.AckTx <- messages.Ack{MessageID: newHA.MessageID, NodeID: node.ID}

				// lets check if I have already received this message, if not its update time!
				if lastHallAssignmentMessageID != newHA.MessageID {
					node.ElevLightAndAssignmentUpdateTx <- makeHallAssignmentAndLightMessage(newHA.HallAssignment, node.GlobalHallRequests)
					lastHallAssignmentMessageID = newHA.MessageID
				}

				// the hall assignments are for me, so I can ack them
				node.AckTx <- messages.Ack{MessageID: newHA.MessageID, NodeID: node.ID}

				// lets check if I have already received this message, if not its update time!
				if lastHallAssignmentMessageID != newHA.MessageID {
					node.ElevLightAndAssignmentUpdateTx <- makeHallAssignmentAndLightMessage(newHA.HallAssignment, node.GlobalHallRequests)
					lastHallAssignmentMessageID = newHA.MessageID
				}
			}


		case newGlobalHallReq := <-node.GlobalHallRequestRx:
			node.TOLC = time.Now()
			masterConnectionTimeoutTimer.Reset(config.MASTER_CONNECTION_TIMEOUT)

			if hasChanged(newGlobalHallReq.HallRequests, node.GlobalHallRequests) {
				node.GlobalHallRequests = newGlobalHallReq.HallRequests
				// fmt.Printf("New global hall request: %v\n", node.GlobalHallRequests)
				node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(newGlobalHallReq.HallRequests)
			}

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

func canAcceptHallAssignments(newHAmsg messages.NewHallAssignments, globalHallReq [config.NUM_FLOORS][2]bool, myID int) bool {
	if newHAmsg.NodeID != myID {
		return false
	} else {
		for floor := range config.NUM_FLOORS {
			// check if my new assignment contains assignments that I am yet to be informed of from master
			if newHAmsg.HallAssignment[floor][elevator.ButtonHallDown] && !(globalHallReq[floor][elevator.ButtonHallDown]) {
				return false
			}
			if newHAmsg.HallAssignment[floor][elevator.ButtonHallUp] && !(globalHallReq[floor][elevator.ButtonHallUp]) {
				return false
			}
		}
	}
	return true
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

func iHaveLostConenction(networkEvent messagehandler.NetworkEvent) bool {
	return networkEvent == messagehandler.NodeHasLostConnection
}
