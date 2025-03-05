package node

import (
	"context"
	"elev/Network/comm"
	"elev/Network/network/bcast"
	"elev/Network/network/messages"
	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"elev/elevatoralgo"
	"elev/util/config"
	"elev/util/msgidbuffer"
	"fmt"
	"strconv"
	"time"

	"github.com/looplab/fsm"
)

type NodeData struct {
	ID        int
	NodeState *fsm.FSM

	GlobalHallRequests [config.NUM_FLOORS][2]bool

	AckTx        chan messages.Ack
	ElevStatesTx chan messages.ElevStates

	HallAssignmentTx  chan messages.NewHallAssignments // send hall assignments to elevators on network
	HallAssignmentsRx chan messages.NewHallAssignments // Receives hall assignments from other nodes

	HallLightUpdateTx chan messages.HallLightUpdate
	HallLightUpdateRx chan messages.HallLightUpdate

	CabRequestInfoTx chan messages.CabRequestInfo
	CabRequestInfoRx chan messages.CabRequestInfo

	GlobalHallRequestTx chan messages.GlobalHallRequest
	GlobalHallRequestRx chan messages.GlobalHallRequest

	ConnectionReqTx    chan messages.ConnectionReq
	ConnectionReqRx    chan messages.ConnectionReq
	ConnectionReqAckRx chan messages.Ack // Receives acknowledgement for request to connect to another node

	commandTx          chan string
	ActiveElevStatesRx chan map[int]messages.ElevStates // Receives the state of the other active node's elevators
	AllElevStatesRx    chan map[int]messages.ElevStates
	TOLCRx             chan time.Time // Receives the Time of Last Contact
	ActiveNodeIDsRx    chan []int     // Receives the IDs of the active nodes on the network

	NewHallReqTx chan messages.NewHallRequest // Sends new hall requests to other nodes
	NewHallReqRx chan messages.NewHallRequest // Receives new hall requests from other nodes

	ElevatorHallButtonEventTx chan elevator.ButtonEvent             // Receives local hall calls from elevator
	ElevatorHallButtonEventRx chan elevator.ButtonEvent             // Receives hall calls from node
	ElevatorHRAStatesRx       chan hallRequestAssigner.HRAElevState // Receives the elevator's HRA states

	HallAssignmentCompleteTx    chan messages.HallAssignmentComplete
	HallAssignmentCompleteRx    chan messages.HallAssignmentComplete
	HallAssignmentCompleteAckRx chan messages.Ack

	GlobalHallReqTransmitEnableTx chan bool
}

func Node(id int) *NodeData {

	node := &NodeData{
		ID: id,
	}
	node.NodeState = fsm.NewFSM(
		"inactive",
		fsm.Events{
			{Name: "initialize", Src: []string{"inactive"}, Dst: "disconnected"},
			{Name: "connect", Src: []string{"disconnected"}, Dst: "slave"},
			{Name: "disconnect", Src: []string{"slave", "master"}, Dst: "disconnected"},
			{Name: "promote", Src: []string{"slave", "disconnected"}, Dst: "master"},
			{Name: "demote", Src: []string{"master"}, Dst: "slave"},
			{Name: "inactivate", Src: []string{"slave", "disconnected", "master"}, Dst: "inactive"},
		},

		fsm.Callbacks{
			"enter_state": func(_ context.Context, e *fsm.Event) {
				fmt.Printf("Node %d skiftet fra %s til %s\node", node.ID, e.Src, e.Dst)
			},

			"enter_master":       node.onEnterMaster,
			"enter_slave":        node.onEnterSlave,
			"enter_disconnected": node.onEnterDisconnected,
			"enter_inactive":     node.onEnterInactive,
		},
	)

	// broadcast channels
	node.AckTx = make(chan messages.Ack)
	node.ElevStatesTx = make(chan messages.ElevStates)
	node.CabRequestInfoTx = make(chan messages.CabRequestInfo) //
	node.ConnectionReqTx = make(chan messages.ConnectionReq)
	node.NewHallReqTx = make(chan messages.NewHallRequest)
	node.HallAssignmentCompleteTx = make(chan messages.HallAssignmentComplete)

	HATransToBcastTx := make(chan messages.NewHallAssignments)         // channel for comm from Hall Assignment Transmitter process to Broadcaster
	lightUpdateTransToBroadcast := make(chan messages.HallLightUpdate) //channel for communication from light update transmitter process and broadcaster
	globalHallReqTransToBroadcast := make(chan messages.GlobalHallRequest)

	// broadcast all messages on channels to udp process
	go bcast.Broadcaster(config.PORT_NUM, node.AckTx, node.ElevStatesTx, HATransToBcastTx, node.CabRequestInfoTx, globalHallReqTransToBroadcast, lightUpdateTransToBroadcast, node.ConnectionReqTx, node.NewHallReqTx, node.HallAssignmentCompleteTx)

	node.HallAssignmentsRx = make(chan messages.NewHallAssignments)
	node.CabRequestInfoRx = make(chan messages.CabRequestInfo)
	node.GlobalHallRequestRx = make(chan messages.GlobalHallRequest)
	node.HallLightUpdateRx = make(chan messages.HallLightUpdate)
	node.ConnectionReqRx = make(chan messages.ConnectionReq)
	node.NewHallReqRx = make(chan messages.NewHallRequest)
	node.HallAssignmentCompleteRx = make(chan messages.HallAssignmentComplete)

	ackRx := make(chan messages.Ack)
	elevStatesRx := make(chan messages.ElevStates)

	go bcast.Receiver(config.PORT_NUM, ackRx, elevStatesRx, node.HallAssignmentsRx, node.CabRequestInfoRx, node.GlobalHallRequestRx, node.HallLightUpdateRx, node.ConnectionReqRx, node.NewHallReqRx, node.HallAssignmentCompleteRx)

	lightUpdateAckRx := make(chan messages.Ack)
	hallAssignmentsAckRx := make(chan messages.Ack)
	node.ConnectionReqAckRx = make(chan messages.Ack)
	node.HallAssignmentCompleteAckRx = make(chan messages.Ack)

	// process for distributing incoming acks in ackRx to different processes
	go comm.IncomingAckDistributor(ackRx, hallAssignmentsAckRx, lightUpdateAckRx, node.ConnectionReqAckRx, node.HallAssignmentCompleteAckRx)

	node.HallAssignmentTx = make(chan messages.NewHallAssignments)
	// process responsible for sending and making sure hall assignments are acknowledged
	go comm.HallAssignmentsTransmitter(HATransToBcastTx, node.HallAssignmentTx, hallAssignmentsAckRx)

	node.ElevatorHallButtonEventTx = make(chan elevator.ButtonEvent)
	node.ElevatorHallButtonEventRx = make(chan elevator.ButtonEvent)
	node.ElevatorHRAStatesRx = make(chan hallRequestAssigner.HRAElevState)
	go elevatoralgo.ElevatorProgram(node.ElevatorHallButtonEventRx, node.ElevatorHRAStatesRx, node.ElevatorHallButtonEventTx)

	node.commandTx = make(chan string)
	node.TOLCRx = make(chan time.Time)
	node.ActiveElevStatesRx = make(chan map[int]messages.ElevStates)
	node.AllElevStatesRx = make(chan map[int]messages.ElevStates)
	node.ActiveNodeIDsRx = make(chan []int)
	go comm.ElevStatesListener(node.ID, node.commandTx, node.TOLCRx, node.ActiveElevStatesRx, node.ActiveNodeIDsRx, elevStatesRx, node.AllElevStatesRx)

	node.GlobalHallRequestTx = make(chan messages.GlobalHallRequest) //
	node.GlobalHallReqTransmitEnableTx = make(chan bool)
	go comm.GlobalHallRequestsTransmitter(node.GlobalHallReqTransmitEnableTx, globalHallReqTransToBroadcast, node.GlobalHallRequestTx)

	node.HallLightUpdateTx = make(chan messages.HallLightUpdate) //
	go comm.LightUpdateTransmitter(lightUpdateTransToBroadcast, node.HallLightUpdateTx, lightUpdateAckRx)

	return node
}

func (node *NodeData) onEnterInactive(_ context.Context, e *fsm.Event) {
	InactiveProgram(node)
}

func (node *NodeData) onEnterDisconnected(_ context.Context, e *fsm.Event) {
	DisconnectedProgram(node)
}

func (node *NodeData) onEnterSlave(_ context.Context, e *fsm.Event) {
	SlaveProgram(node)
}

func (node *NodeData) onEnterMaster(_ context.Context, e *fsm.Event) {
	MasterProgram(node)
}

func InactiveProgram(node *NodeData) {
	fmt.Printf("Node %d is now Inactive\n", node.ID)
	if err := node.NodeState.Event(context.Background(), "initialize"); err != nil {
		fmt.Println("Error:", err)
	}
}

func DisconnectedProgram(node *NodeData) {
	fmt.Printf("Node %d is now Disconnected\n", node.ID)
	timeOfLastContact := time.Time{}                        // placeholder for getting from server
	msgID, _ := comm.GenerateMessageID(comm.CONNECTION_REQ) // placeholder for using "getmessageid function"

	myConnReq := messages.ConnectionReq{TOLC: timeOfLastContact, NodeID: node.ID, MessageID: msgID}
	incomingConnRequests := make(map[int]messages.ConnectionReq)

	// ID of the node we currently are trying to connect with
	currentFriendID := 0

	var lastReceivedAck *messages.Ack

	for {
		select {
		case <-node.GlobalHallRequestRx:
			// here, we must check if the master knows anything about us
			// this message transaction should be defined better than it is now, who sends what?
			if err := node.NodeState.Event(context.Background(), "connect"); err != nil {
				fmt.Println("Error:", err)
			} else {
				return
			}

		case incomingConnReq := <-node.ConnectionReqRx:
			if node.ID != incomingConnReq.NodeID {
				incomingConnRequests[incomingConnReq.NodeID] = incomingConnReq
				if currentFriendID == 0 || currentFriendID > incomingConnReq.NodeID {
					// this is the node with the lowest ID, I want to start a relationship with him
					currentFriendID = incomingConnReq.NodeID
				}
			}

		case connReqAck := <-node.ConnectionReqAckRx:
			if node.ID != connReqAck.NodeID && connReqAck.NodeID == currentFriendID {
				// All these decisions should be moved into a pure function, and the result returned
				// check who has the most recent data
				// here, we must ask on node.commandTx "getTOLC". Then, on return from node.TOLCRx compare
				lastReceivedAck = &connReqAck
				node.commandTx <- "getTOLC"
			}
		case TOLC := <-node.TOLCRx:
			if lastReceivedAck != nil && node.ID != lastReceivedAck.NodeID && lastReceivedAck.NodeID == currentFriendID {
				if connReq, exists := incomingConnRequests[lastReceivedAck.NodeID]; exists {
					shouldBeMaster := ShouldBeMaster(node.ID, lastReceivedAck.NodeID, currentFriendID, TOLC, connReq.TOLC)
					if shouldBeMaster {
						if err := node.NodeState.Event(context.Background(), "promote"); err != nil {
							fmt.Println("Error:", err)
						}
					} else {
						if err := node.NodeState.Event(context.Background(), "connect"); err != nil {
							fmt.Println("Error:", err)
						}
					}
				}
				lastReceivedAck = nil
			}

			// timeout should be a const variable
		case <-time.After(time.Millisecond * 500):
			// start sending a conn request :)
			// isConnRequestActive = true
			node.ConnectionReqTx <- myConnReq
		}
	}
}

func ShouldBeMaster(myID int, otherID int, _currentFriendID int, TOLC time.Time, otherTOLC time.Time) bool {
	// Compare TOLC values to determine who becomes master
	if TOLC.Before(otherTOLC) { // The other node has more recent data --> We should be master
		return true
	} else if TOLC.After(otherTOLC) { // We have more recent data --> We should be slave
		return false
	} else { // TOLC values are equal --> Compare node IDs
		if myID > otherID {
			return true
		} else if myID < otherID {
			return false
		}
	}
}

func SlaveProgram(node *NodeData) {
	fmt.Printf("Node %d is now a Slave\n", node.ID)

	for {
		select {
		case hallAssignment := <-node.HallAssignmentsRx:
			fmt.Printf("Node %d received hall assignment: %v\n", node.ID, hallAssignment)
		case <-time.After(config.MASTER_TIMEOUT):
			fmt.Printf("Node %d alive\n", node.ID)
		}

	}
}

func MasterProgram(node *NodeData) {
	fmt.Printf("Node %d is now a Master\n", node.ID)
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
			activeReq = true
			node.commandTx <- "getActiveElevStates"

		case newElevStates := <-node.ActiveElevStatesRx:
			fmt.Printf("Node %d received active elev states: %v\n", node.ID, newElevStates)
			if activeReq {
				HRAoutput := hallRequestAssigner.HRAalgorithm(newElevStates, node.GlobalHallRequests)
				fmt.Printf("Node %d HRA output: %v\n", node.ID, HRAoutput)
				for id, hallRequests := range *HRAoutput {
					nodeID, err := strconv.Atoi(id)
					if err != nil {
						fmt.Println("Error: ", err)
					}
					fmt.Printf("Node %d sending hall requests to node %d: %v\n", node.ID, nodeID, hallRequests)
					//sending hall requests to all nodes assuming all
					//nodes are connected nad not been disconnected after sending out internal states
					node.HallAssignmentTx <- messages.NewHallAssignments{NodeID: nodeID, HallAssignment: hallRequests, MessageID: 0}
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
						cabRequestInfo = messages.CabRequestInfo{CabRequest: states.CabRequest, ReceiverNodeID: id}
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
		case <-node.ElevatorHallButtonEventRx:
		case <-node.ElevatorHRAStatesRx:
		case <-node.AllElevStatesRx:
		case <-node.TOLCRx:
		case <-node.ActiveNodeIDsRx:
		}
	}
}
