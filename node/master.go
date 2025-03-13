package node

import (
	"elev/Network/messages"
	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"fmt"
)

const bufferSize = 5

// A buffer that holds the last #buffersize message ids
type MessageIDBuffer struct {
	messageIDs [bufferSize]uint64
	size       int
	index      int
}

// using Add, you can add a message ID to the buffer. It overwrites in a FIFO manner
func (buf *MessageIDBuffer) Add(id uint64) {
	if buf.size == buf.index {
		buf.index = 0
	}
	buf.messageIDs[buf.index] = id
	buf.index += 1
}

// check if a message id is in the buffer
func (buf *MessageIDBuffer) Contains(id uint64) bool {
	for i := 0; i < buf.size; i++ {
		if buf.messageIDs[i] == id {
			return true
		}
	}
	return false
}

func MasterProgram(node *NodeData) nodestate {
	fmt.Printf("Node %d is now a Master\n", node.ID)

	var myElevState messages.NodeElevState
	activeNewHallReq := false
	activeConnReq := make(map[int]messages.ConnectionReq)

	var recentHACompleteBuffer MessageIDBuffer
	var nextNodeState nodestate

	node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
	node.GlobalHallReqTransmitEnableTx <- true // start transmitting global hall requests (this means you are a master)
	node.commandToServerTx <- "startConnectionTimeoutDetection"

ForLoop:
	for {
	Select:
		select {
		case elevMsg := <-node.FromElevator:
			switch elevMsg.Type {
			case messages.MsgDoorStuck:
				if elevMsg.IsDoorStuck {
					fmt.Println("Door is stuck, exiting master")
					nextNodeState = Inactive
					break ForLoop
				}

			case messages.MsgHallButtonEvent:
				// Handle hall button event
				fmt.Printf("Node %d received a new hall request from my elevator: %v\n", node.ID, elevMsg.ButtonEvent)
				switch elevMsg.ButtonEvent.Button {
				case elevator.ButtonHallUp:
					node.GlobalHallRequests[elevMsg.ButtonEvent.Floor][elevator.ButtonHallUp] = true
				case elevator.ButtonHallDown:
					node.GlobalHallRequests[elevMsg.ButtonEvent.Floor][elevator.ButtonHallDown] = true
				case elevator.ButtonCab:
					fmt.Println("Received a hall request, but the button type was invalid")
					break Select
				}

				fmt.Printf("New Global hall requests: %v\n", node.GlobalHallRequests)
				activeNewHallReq = true
				node.commandToServerTx <- "getActiveElevStates"

			case messages.MsgHallAssignmentComplete:
				// Handle completed hall assignments
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
				// Update and broadcast elevator state
				node.ElevStatesTx <- messages.NodeElevState{
					NodeID:    node.ID,
					ElevState: elevMsg.ElevState,
				}
			}
		case newHallReq := <-node.NewHallReqRx:

			fmt.Printf("Node %d received a new hall request: %v\n", node.ID, newHallReq)
			switch newHallReq.HallButton {

			case elevator.ButtonHallUp:
				node.GlobalHallRequests[newHallReq.Floor][elevator.ButtonHallUp] = true

			case elevator.ButtonHallDown:
				node.GlobalHallRequests[newHallReq.Floor][elevator.ButtonHallDown] = true

			case elevator.ButtonCab:
				fmt.Println("Received a new hall requests, but the button type was invalid")
				break Select
			}

			fmt.Printf("New Global hall requests: %v\n", node.GlobalHallRequests)
			activeNewHallReq = true

			// send the global hall requests to the server for broadcast to update other nodes
			node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
			node.commandToServerTx <- "getActiveElevStates"

		case completedHallReq := <-node.HallAssignmentCompleteRx:
			if completedHallReq.HallButton != elevator.ButtonCab {
				// Notify other nodes that the hall request/assignment is completed
				hallAssignmentCompleteMsg := messages.HallAssignmentComplete{
					Floor:      completedHallReq.Floor,
					HallButton: completedHallReq.HallButton,
					MessageID:  completedHallReq.MessageID,
				}

				node.HallAssignmentCompleteTx <- hallAssignmentCompleteMsg
				fmt.Printf("Node %d sent hall assignment complete message\n", node.ID)

				node.GlobalHallRequests[completedHallReq.Floor][completedHallReq.HallButton] = false
				node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
			}

		case newElevStates := <-node.ActiveElevStatesFromServerRx:
			if activeNewHallReq {

				newElevStates[node.ID] = myElevState

				fmt.Printf("Node %d received active elev states: %v\n", node.ID, newElevStates)

				// check that the floor is valid - can be its own function
				for id := range newElevStates {
					if newElevStates[id].ElevState.Floor < 0 {
						fmt.Printf("Error: invalid elevator floor for elevator %d ", id)
						break Select
					}
				}

				HRAoutput := hallRequestAssigner.HRAalgorithm(newElevStates, node.GlobalHallRequests)

				fmt.Printf("Node %d HRA output: %v\n", node.ID, HRAoutput)

				for id, hallRequests := range HRAoutput {

					if id == node.ID {
						// here, we must update the lights of our own elevator
						node.ToElevator <- messages.NodeToElevatorMsg{
							Type:            messages.MsgHallAssignment,
							HallAssignments: hallRequests,
						}
						fmt.Printf("Node %d has hall assignment task queue: %v\n", node.ID, hallRequests)

					} else {
						fmt.Printf("Node %d sending hall requests to node %d: %v\n", node.ID, id, hallRequests)

						// distribute the orders!
						node.HallAssignmentTx <- messages.NewHallAssignments{NodeID: id, HallAssignment: hallRequests, MessageID: 0}

					}

				}
				// update the transmitter with the latest global hall requests
				node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
				activeNewHallReq = false
			}

		case connReq := <-node.ConnectionReqRx:
			// here, there may need to be some extra logic
			if connReq.TOLC.IsZero() {
				activeConnReq[connReq.NodeID] = connReq
				node.commandToServerTx <- "getAllElevStates"
			}

		case allElevStates := <-node.AllElevStatesFromServerRx:
			if len(activeConnReq) != 0 {

				for id := range activeConnReq {
					var cabRequestInfo messages.CabRequestInfo
					if states, ok := allElevStates[id]; ok {
						cabRequestInfo = messages.CabRequestInfo{CabRequest: states.ElevState.CabRequests, ReceiverNodeID: id}
					}
					// this message may not arrive. If the disconnected node waits for its arrival, that means it will never become a slave
					node.CabRequestInfoTx <- cabRequestInfo
					delete(activeConnReq, id)
				}
			}

		case HA := <-node.HallAssignmentCompleteRx:

			// check that this is not a message you have already received
			if !recentHACompleteBuffer.Contains(HA.MessageID) {

				if HA.HallButton != elevator.ButtonCab {
					node.GlobalHallRequests[HA.Floor][HA.HallButton] = false
				} else {
					fmt.Printf("Received invalid completed hall assignment complete message, completion %v", HA.HallButton)
				}

				recentHACompleteBuffer.Add(HA.MessageID)

				// update the transmitter with the newest global hall requests
				node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}

			}

			node.AckTx <- messages.Ack{MessageID: HA.MessageID, NodeID: node.ID}

		case timeout := <-node.ConnectionLossEventRx:
			if timeout {
				fmt.Println("Connection timed out, exiting master")

				nextNodeState = Disconnected
				break ForLoop
			}

		case <-node.HallAssignmentsRx:
		case <-node.HallAssignmentCompleteAckRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
		case <-node.HallLightUpdateRx:
		case <-node.ConnectionReqAckRx:
		case <-node.AllElevStatesFromServerRx:
		case <-node.TOLCFromServerRx:
		case <-node.ActiveNodeIDsFromServerRx:
			// when you get a message on any of these channels, do nothing
		}
	}
	node.GlobalHallReqTransmitEnableTx <- false // stop transmitting global hall requests
	return nextNodeState
}
