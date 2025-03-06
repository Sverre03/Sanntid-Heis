package node

import (
	"context"
	"elev/Network/network/messages"
	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"elev/util/config"
	"elev/util/msgidbuffer"
	"fmt"
	"strconv"
	"time"
)

func MasterProgram(node *NodeData) {
	fmt.Printf("Node %d is now a Master\n", node.ID)
	var myCurrentState messages.NodeElevState
	activeReq := false
	activeConnReq := make(map[int]messages.ConnectionReq) // do we need an ack on this
	var recentHACompleteBuffer msgidbuffer.MessageIDBuffer

	node.GlobalHallReqTransmitEnableTx <- true // start transmitting global hall requests (this means you are a master)

	for {
		select {
		case newHallReq := <-node.NewHallReqRx:
			fmt.Printf("Node %d received a new hall request: %v\n", node.ID, newHallReq)
			switch newHallReq.HallButton {

			case elevator.BT_HallUp:
				node.GlobalHallRequests[newHallReq.Floor][elevator.BT_HallUp] = true

			case elevator.BT_HallDown:
				node.GlobalHallRequests[newHallReq.Floor][elevator.BT_HallDown] = true

			case elevator.BT_Cab:
				fmt.Println("Received a new hall requests, but the button type was invalid")
			}

			fmt.Printf("New Global hall requests: %v\n", node.GlobalHallRequests)
			activeReq = true
			node.commandTx <- "getActiveElevStates"

		case newElevStates := <-node.ActiveElevStatesRx:
			newElevStates[node.ID] = myCurrentState
			fmt.Printf("Node %d received active elev states: %v\n", node.ID, newElevStates)

			for id := range newElevStates {
				if newElevStates[id].ElevState.Floor < 0 {
					fmt.Println("Error: invalid elevator floor for elevator %d ", id)
					return
				}
			}
			if activeReq {
				HRAoutput := hallRequestAssigner.HRAalgorithm(newElevStates, node.GlobalHallRequests)
				fmt.Printf("Node %d HRA output: %v\n", node.ID, HRAoutput)
				for id, hallRequests := range *HRAoutput {
					nodeID, err := strconv.Atoi(id)
					if err != nil {
						fmt.Println("Error: ", err)
					}
					if nodeID == node.ID {
						hallAssignmentTaskQueue := hallRequests
						fmt.Printf("Node %d has hall assignment task queue: %v\n", node.ID, hallAssignmentTaskQueue)
					}else{
						fmt.Printf("Node %d sending hall requests to node %d: %v\n", node.ID, nodeID, hallRequests)
						//sending hall requests to all nodes assuming all
						//nodes are connected and not been disconnected after sending out internal states
						node.HallAssignmentTx <- messages.NewHallAssignments{NodeID: nodeID, HallAssignment: hallRequests, MessageID: 0}
					}

					
				}
				node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}
				activeReq = false
			}

		case connReq := <-node.ConnectionReqRx:
			// here, there may need to be some extra logic
			if connReq.TOLC.IsZero() {
				activeConnReq[connReq.NodeID] = connReq
				node.commandTx <- "getAllElevStates"
			}

		case allElevStates := <-node.AllElevStatesRx:
			if len(activeConnReq) != 0 {

				//If activeConnectionReq is true, send all activeElevStates to nodes in activeConnReq

				// her antas det at en id eksisterer i allElevStates Mappet dersom den eksisterer i activeConnReq, dette er en feilaktig antagelse

				for id := range activeConnReq {
					var cabRequestInfo messages.CabRequestInfo
					if states, ok := allElevStates[id]; ok {
						cabRequestInfo = messages.CabRequestInfo{CabRequest: states.ElevState.CabRequests, ReceiverNodeID: id}
					}
					// sjekke om id finnes i map
					// hvis ja: send svar
					// hvis nei: send svar likevel
					node.CabRequestInfoTx <- cabRequestInfo
					delete(activeConnReq, id)
				}
			}

		case HA := <-node.HallAssignmentCompleteRx:
			// this logic could go somewhere else to clean up the master program
			if !recentHACompleteBuffer.Contains(HA.MessageID) {

				// in case ButtonType is not hall button, this line of code will crash the program!
				if HA.HallButton != elevator.BT_Cab {
					node.GlobalHallRequests[HA.Floor][HA.HallButton] = false
				} else {
					fmt.Println("Some less intelligent cretin sent a hall assignment complete message with the wrong button type (cab btn)")
				}

				recentHACompleteBuffer.Add(HA.MessageID)
				// update the transmitter with the newest global hall requests
				node.GlobalHallRequestTx <- messages.GlobalHallRequest{HallRequests: node.GlobalHallRequests}

			}

			node.AckTx <- messages.Ack{MessageID: HA.MessageID, NodeID: node.ID}

		case <-node.HallAssignmentsRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
		case <-node.HallLightUpdateRx:
		case <-node.ConnectionReqAckRx:
		case <-node.ElevatorHRAStatesRx:
		case <-node.AllElevStatesRx:
		case <-node.TOLCRx:
		case <-node.ActiveNodeIDsRx:
		//Master running its own elevator
		case isDoorStuck := <-node.IsDoorStuckCh:
			if isDoorStuck {
				if err := node.NodeState.Event(context.Background(), "inactivate"); err != nil {
					fmt.Println("Error:", err)
				}
			}
		case <-time.After(config.NODE_DOOR_POLL_RATE):
			node.RequestDoorStateCh <- true

		case currentElevStates := <-node.ElevatorHRAStatesRx:
			myCurrentState = messages.NodeElevState{NodeID: node.ID, ElevState: currentElevStates}
			node.ElevStatesTx <- messages.NodeElevState{NodeID: node.ID, ElevState: currentElevStates}

		case newHallReq := <-node.ElevatorHallButtonEventRx:
			fmt.Printf("Node %d received a new hall request: %v\n", node.ID, newHallReq)
			switch newHallReq.Button {

			case elevator.BT_HallUp:
				node.GlobalHallRequests[newHallReq.Floor][elevator.BT_HallUp] = true

			case elevator.BT_HallDown:
				node.GlobalHallRequests[newHallReq.Floor][elevator.BT_HallDown] = true

			case elevator.BT_Cab:
				fmt.Println("Received a new hall requests, but the button type was invalid")
			}

			fmt.Printf("New Global hall requests: %v\n", node.GlobalHallRequests)
			activeReq = true
			node.commandTx <- "getActiveElevStates"
		}
	}
}
