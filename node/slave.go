package node

import (
	"elev/config"
	"elev/elevator"
	"elev/network/communication"
	"elev/network/messages"
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

	select {
	case node.commandToServerTx <- "startConnectionTimeoutDetection":
	default:
		fmt.Printf("Warning: Command channel is full, command %s not sent\n", "startConnectionTimeoutDetection")
	}

ForLoop:
	for {
		select {
		case elevMsg := <-node.ElevatorEventRx:
			switch elevMsg.EventType {
			case singleelevator.ElevStatusUpdateEvent:
				if elevMsg.ElevIsDown {
					nextNodeState = Inactive
					break ForLoop
				}

			case singleelevator.HallButtonEvent:
				node.NewHallReqTx <- makeNewHallReq(node.ID, elevMsg)
			}

		case myElevStates := <-node.MyElevStatesRx:
			// Broadcast elevator states to network
			node.NodeElevStatesTx <- messages.NodeElevState{
				NodeID:    node.ID,
				ElevState: myElevStates,
			}

		case networkEvent := <-node.NetworkEventRx:
			// check if we have lost connection
			if networkEvent == communication.NodeHasLostConnection {
				nextNodeState = Disconnected
				break ForLoop
			}

		case newHA := <-node.HallAssignmentsRx:
			if canAcceptHallAssignments(newHA, node.GlobalHallRequests, node.ID) {
				// the hall assignments are for me, so I can ack them
				node.AckTx <- messages.Ack{MessageID: newHA.MessageID, NodeID: node.ID}

				// lets check if I have already received this message, if not its update time!
				if lastHallAssignmentMessageID != newHA.MessageID {
					node.ElevLightAndAssignmentUpdateTx <- makeHallAssignmentAndLightMessage(newHA.HallAssignment, node.GlobalHallRequests, newHA.HallAssignmentCounter)
					lastHallAssignmentMessageID = newHA.MessageID
				}
			}

		case newGlobalHallReq := <-node.GlobalHallRequestRx:
			// We received global hall requests from master which means master is alive, reset timer and update my contact counter
			node.ContactCounter = newGlobalHallReq.CounterValue
			masterConnectionTimeoutTimer.Reset(config.MASTER_CONNECTION_TIMEOUT)
			
			// If global hall requests has changed update lights
			if hasChanged(node.GlobalHallRequests, newGlobalHallReq.HallRequests) {
				node.GlobalHallRequests = newGlobalHallReq.HallRequests
				node.ElevLightAndAssignmentUpdateTx <- makeLightMessage(newGlobalHallReq.HallRequests)
			}

		case <-masterConnectionTimeoutTimer.C:
			// I havent received anything from master in a given time period, so change state to disconnected
			nextNodeState = Disconnected
			break ForLoop

		case <-node.ElevStateUpdatesFromServer:
		case <-node.ConnectionReqRx:
		case <-node.CabRequestInfoRx:
		case <-node.NewHallReqRx:
			// read these to prevent blocking
		}

	}

	select {
	case node.commandToServerTx <- "stopConnectionTimeoutDetection":
	default:
		fmt.Printf("Warning: Command channel is full, command %s not sent\n", "stopConnectionTimeoutDetection")
	}

	return nextNodeState
}

func canAcceptHallAssignments(newHAmsg messages.NewHallAssignments, globalHallReq [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool, myID int) bool {
	if newHAmsg.NodeID != myID {
		return false
	} else {
		// check if my new assignment contains assignments that I am yet to be informed of from master
		// this means that the lights are not yet lit for the assignment
		for floor := range config.NUM_FLOORS {
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

func hasChanged(oldGlobalHallReq, newGlobalHallReq [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool) bool {
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
