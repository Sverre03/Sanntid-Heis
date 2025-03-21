package node

import (
	"elev/Network/messagehandler"
	"elev/Network/messages"
	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"elev/singleelevator"
	"elev/util/config"
	"fmt"
	"time"
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
	shouldDistributeHallRequests := false
	activeConnReq := make(map[int]messages.ConnectionReq)

	var recentHACompleteBuffer MessageIDBuffer
	var nextNodeState nodestate

	// inform the global hall request transmitter of the new global hall requests
	node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}

	// start the transmitters
	node.GlobalHallReqTransmitEnableTx <- true
	node.HallRequestAssignerTransmitEnableTx <- true
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

				break Select

			case singleelevator.HallButtonEvent:
				// new hallbuttonpress from my elevator!
				if elevMsg.ButtonEvent.Button != elevator.ButtonCab {
					node.GlobalHallRequests[elevMsg.ButtonEvent.Floor][elevMsg.ButtonEvent.Button] = true
					shouldDistributeHallRequests = true
				}

			case singleelevator.LocalHallAssignmentCompleteEvent:
				// update the global hall assignments
				if elevMsg.ButtonEvent.Button != elevator.ButtonCab {
					node.GlobalHallRequests[elevMsg.ButtonEvent.Floor][elevMsg.ButtonEvent.Button] = false
				}
			}

			if shouldDistributeHallRequests {
				fmt.Printf("New Global hall requests: %v\n", node.GlobalHallRequests)
				node.commandToServerTx <- "getActiveElevStates"
			}
			// update the hall request transmitter with the newest requests
			node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}

		case myStates := <-node.MyElevStatesRx:
			myElevState = messages.NodeElevState{NodeID: node.ID, ElevState: myStates}
			node.NodeElevStatesTx <- myElevState

		case newHallReq := <-node.NewHallReqRx:

			fmt.Printf("Node %d received a new hall request: %v\n", node.ID, newHallReq)
			if newHallReq.HallButton == elevator.ButtonCab {
				fmt.Println("Received a new hall requests, but the button type was invalid")
				break Select
			}

			node.GlobalHallRequests[newHallReq.Floor][newHallReq.HallButton] = true
			fmt.Printf("New Global hall requests: %v\n", node.GlobalHallRequests)
			shouldDistributeHallRequests = true

			// send the global hall requests to the server for broadcast to update other nodes
			node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
			node.commandToServerTx <- "getActiveElevStates"

		case elevStatesUpdate := <-node.NodeElevStateUpdate:
			if shouldDistributeHallRequests && elevStatesUpdate.OnlyActiveNodes { // We have received state from all active nodes

				// add my elevator to the list of active elevators
				elevStatesUpdate.NodeElevStatesMap[node.ID] = myElevState.ElevState

				fmt.Printf("Node %d received active elev states: %v\n", node.ID, elevStatesUpdate)

				HRAoutput := hallRequestAssigner.HRAalgorithm(elevStatesUpdate.NodeElevStatesMap, node.GlobalHallRequests)

				fmt.Printf("Node %d HRA output: %v\n", node.ID, HRAoutput)

				for id, hallRequests := range HRAoutput {

					if id == node.ID {
						// the message belongs to our elevator
						node.ElevAssignmentLightUpdateTx <- makeHallAssignmentAndLightMessage(hallRequests, node.GlobalHallRequests)
						fmt.Printf("Node %d has hall assignment task queue: %v\n", node.ID, hallRequests)

					} else {
						fmt.Printf("Node %d sending hall requests to node %d: %v\n", node.ID, id, hallRequests)

						// distribute the orders!
						node.HallAssignmentTx <- messages.NewHallAssignments{NodeID: id, HallAssignment: hallRequests, MessageID: 0}

					}

				}
				// update the transmitter with the latest global hall requests
				node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
				shouldDistributeHallRequests = false
			}

			if !elevStatesUpdate.OnlyActiveNodes {

				for id := range activeConnReq {
					var cabRequestInfo messages.CabRequestInfo
					if states, ok := elevStatesUpdate.NodeElevStatesMap[id]; ok {
						cabRequestInfo = messages.CabRequestInfo{CabRequest: states.CabRequests, ReceiverNodeID: id}
					} else {

						// we still send a message, but just with false
						emptySlice := [config.NUM_FLOORS]bool{}

						cabRequestInfo = messages.CabRequestInfo{CabRequest: emptySlice, ReceiverNodeID: id}
					}

					node.CabRequestInfoTx <- cabRequestInfo
					delete(activeConnReq, id)
				}
			}

		case connReq := <-node.ConnectionReqRx:
			if connReq.TOLC.IsZero() {
				activeConnReq[connReq.NodeID] = connReq
				node.commandToServerTx <- "getAllElevStates"
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
			// ack the message, as if we have received it before our previous ack did not arrive
			node.AckTx <- messages.Ack{MessageID: HA.MessageID, NodeID: node.ID}

		case networkEvent := <-node.NetworkEventRx:
			if networkEvent == messagehandler.NodeHasLostConnection {
				fmt.Println("Connection timed out, exiting master")

				nextNodeState = Disconnected
				break ForLoop
			} else if networkEvent == messagehandler.NodeConnectDisconnect {
				shouldDistributeHallRequests = true
				node.commandToServerTx <- "getActiveElevStates"
			}

		case <-node.HallAssignmentsRx:
		case <-node.HallAssignmentCompleteAckRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
		case <-node.ConnectionReqAckRx:

			// when you get a message on any of these channels, do nothing

		}
	}

	// stop transmitters
	node.GlobalHallReqTransmitEnableTx <- false
	node.HallRequestAssignerTransmitEnableTx <- false
	node.TOLC = time.Now()

	return nextNodeState
}
