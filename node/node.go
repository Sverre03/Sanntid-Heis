package node

import (
	"context"
	"elev/Network/comm"
	"elev/Network/network/bcast"
	"elev/Network/network/messages"
	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"elev/util/config"
	"elev/util/msgid_buffer"
	"fmt"
	"strconv"
	"time"

	"github.com/looplab/fsm"
)

type NodeData struct {
	ID        int
	NodeState *fsm.FSM

	TOLC                      time.Time
	Elevator                  *elevator.Elevator
	TaskQueue                 []string
	GlobalHallRequests        []string
	LastKnownStatesOfAllNodes map[int]string

	AckTx chan messages.Ack

	ElevStatesTx chan messages.ElevStates

	HallAssignmentsRx       chan messages.NewHallAssignments
	OutGoingHallAssignments chan messages.NewHallAssignments

	CabRequestInfoRx chan messages.CabRequestINF

	GlobalHallRequestRx chan messages.GlobalHallRequest

	HallLightUpdateRx chan messages.HallLightUpdate

	ConnectionReqTx chan messages.ConnectionReq
	ConnectionReqRx chan messages.ConnectionReq

	commandCh          chan string
	ActiveElevStatesRx chan map[int]messages.ElevStates
	AllElevStatesRx    chan map[int]messages.ElevStates
	TOLCRx             chan time.Time
	ActiveNodeIDsRx    chan []int

	ConnectionReqAckRx chan messages.Ack

	NewHallReqTx chan messages.NewHallRequest // Sends new hall requests to other nodes
	NewHallReqRx chan messages.NewHallRequest // Receives new hall requests from other nodes

	ElevatorHallButtonEventTx chan elevator.ButtonEvent // Receives local hall calls from elevator
	ElevatorHallButtonEventRx chan elevator.ButtonEvent // Receives hall calls from node

	ElevatorHRAStatesRx chan hallRequestAssigner.HRAElevState

	HallAssignmentCompleteRx chan messages.HallAssignmentComplete

	transmitEnableCh chan bool

}

func Node(id int) *NodeData {

	node := &NodeData{
		ID:                        id,
		TOLC:                      time.Time{},
		Elevator:                  &elevator.Elevator{},
		TaskQueue:                 make([]string, 0),
		GlobalHallRequests:        make([]string, 0),
		LastKnownStatesOfAllNodes: make(map[int]string),
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

	node.AckTx = make(chan messages.Ack)
	AckRx := make(chan messages.Ack) //

	node.ElevStatesTx = make(chan messages.ElevStates)
	ElevStatesRx := make(chan messages.ElevStates) //

	HallAssignmentsTx := make(chan messages.NewHallAssignments)
	node.HallAssignmentsRx = make(chan messages.NewHallAssignments)
	node.OutGoingHallAssignments = make(chan messages.NewHallAssignments)

	CabRequestInfoTx := make(chan messages.CabRequestINF) //
	node.CabRequestInfoRx = make(chan messages.CabRequestINF)

	GlobalHallRequestTx := make(chan messages.GlobalHallRequest) //
	node.GlobalHallRequestRx = make(chan messages.GlobalHallRequest)

	HallLightUpdateTx := make(chan messages.HallLightUpdate) //
	node.HallLightUpdateRx = make(chan messages.HallLightUpdate)

	node.ConnectionReqTx = make(chan messages.ConnectionReq)
	node.ConnectionReqRx = make(chan messages.ConnectionReq)

	node.NewHallReqTx = make(chan messages.NewHallRequest)
	node.NewHallReqRx = make(chan messages.NewHallRequest)

	HallAssignmentCompleteTx := make(chan messages.HallAssignmentComplete) //
	node.HallAssignmentCompleteRx = make(chan messages.HallAssignmentComplete)

	HallAssignmentsAckTx := make(chan messages.Ack)

	node.commandCh = make(chan string)
	node.TOLCRx = make(chan time.Time)

	node.ActiveElevStatesRx = make(chan map[int]messages.ElevStates)
	node.AllElevStatesRx = make(chan map[int]messages.ElevStates)
	node.ActiveNodeIDsRx = make(chan []int)

	node.transmitEnableCh = make(chan bool)

	elevStatesRx := make(chan messages.ElevStates)

	go bcast.Transmitter(config.PORT_NUM, node.AckTx, node.ElevStatesTx, HallAssignmentsTx, CabRequestInfoTx, GlobalHallRequestTx, HallLightUpdateTx, node.ConnectionReqTx, node.NewHallReqTx, HallAssignmentCompleteTx)
	go bcast.Receiver(config.PORT_NUM, AckRx, ElevStatesRx, node.HallAssignmentsRx, node.CabRequestInfoRx, node.GlobalHallRequestRx, node.HallLightUpdateRx, node.ConnectionReqRx, node.NewHallReqRx, node.HallAssignmentCompleteRx)
	go comm.HallAssignmentsTransmitter(HallAssignmentsTx, node.OutGoingHallAssignments, HallAssignmentsAckTx)
	go comm.ElevStatesListener(node.commandCh, node.TOLCRx, node.ActiveElevStatesRx, node.ActiveNodeIDsRx, elevStatesRx, node.AllElevStatesRx)
	go comm.GlobalHallRequestsTransmitter(node.transmitEnableCh, GlobalHallRequestTx, node.GlobalHallRequestRx)
	return node
}

func (node *NodeData) onEnterInactive(_ context.Context, e *fsm.Event) {
	fmt.Printf("Node %d er nå INACTIVE. Med TOLC lik %s \node", node.ID, node.TOLC)
	InactiveProgram(node)
}

func (node *NodeData) onEnterDisconnected(_ context.Context, e *fsm.Event) {
	node.TOLC = time.Time{}
	fmt.Printf("Node %d er nå DISCONNECTED. Med TOLC lik %s \node", node.ID, node.TOLC)
	DisconnectedProgram(node)
}

func (node *NodeData) onEnterSlave(_ context.Context, e *fsm.Event) {
	node.TOLC = time.Now()
	// fmt.Printf("Node %d er nå SLAVE. Med TOLC lik %s \node", node.ID, node.TOLC)
	SlaveProgram(node)
}

func (node *NodeData) onEnterMaster(_ context.Context, e *fsm.Event) {
	node.TOLC = time.Now()
	// fmt.Printf("Node %d er nå MASTER. Med TOLC lik %s \node", node.ID, node.TOLC)
	MasterProgram(node)
}

func InactiveProgram(node *NodeData) {
	if err := node.NodeState.Event(context.Background(), "initialize"); err != nil {
		fmt.Println("Error:", err)
	}
}

func DisconnectedProgram(node *NodeData) {
	timeOfLastContact := time.Time{}                        // placeholder for getting from server
	msgID, _ := comm.GenerateMessageID(comm.CONNECTION_REQ) // placeholder for using "getmessageid function"

	myConnReq := messages.ConnectionReq{TOLC: timeOfLastContact, NodeID: node.ID, MessageID: msgID}
	incomingConnRequests := make(map[int]messages.ConnectionReq)

	// ID of the node we currently are trying to connect with
	currentFriendID := 0

	// isConnRequestActive := false

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
				if node.TOLC.Before(incomingConnRequests[connReqAck.NodeID].TOLC) {
					if err := node.NodeState.Event(context.Background(), "promote"); err != nil {
						fmt.Println("Error:", err)
					}

				} else if node.TOLC.After(incomingConnRequests[connReqAck.NodeID].TOLC) {
					if err := node.NodeState.Event(context.Background(), "connect"); err != nil {
						fmt.Println("Error:", err)
					}

				} else {
					// tie breaker: the one with the largeest ID becomes the master
					if node.ID > connReqAck.NodeID {
						if err := node.NodeState.Event(context.Background(), "promote"); err != nil {
							fmt.Println("Error:", err)
						}
					} else if node.ID < connReqAck.NodeID {
						if err := node.NodeState.Event(context.Background(), "connect"); err != nil {
							fmt.Println("Error:", err)
						}
					}
				}
			}

			// timeout should be a const variable
		case <-time.After(time.Millisecond * 500):

			// start sending a conn request :)
			// isConnRequestActive = true
			node.ConnectionReqTx <- myConnReq
		}
	}
}

func SlaveProgram(node *NodeData) {
	fmt.Printf("Node %d is now a slave\n", node.ID)
	for{
		select {
		case <- time.After(config.MASTER_TIMEOUT):
			fmt.Printf("Node %d info sent\n", node.ID)
			node.NewHallReqTx <- messages.NewHallRequest{Floor: 1, HallButton: elevator.BT_HallUp}
			node.ElevStatesTx <- messages.ElevStates{NodeID: node.ID, Direction: elevator.MD_Up, Floor: 1, CabRequest: [config.NUM_FLOORS]bool{false, false, false, false}, Behavior: "idle"}
		case hallAssignment := <- node.HallAssignmentsRx:
			fmt.Printf("Node %d received hall assignment: %v\n", node.ID, hallAssignment)
		}
	}
}

func MasterProgram(node *NodeData) {
	fmt.Printf("Node %d is now a master\n", node.ID)
	activeReq := false
	activeConnectionReq := false
	var allElevStates map[int]messages.ElevStates
	var activeHallRequests [config.NUM_FLOORS][2]bool     //Get activeHallRequests from previous master saved in server if existing
	activeConnReq := make(map[int]messages.ConnectionReq) // do we need an ack on this
	var recentHACompleteBuffer msgid_buffer.MessageIDBuffer

	for i := 0; i < config.NUM_FLOORS; i++ {  //Emtpy activeHallRequests list if no activeHallRequests from previous master, placeholder for now
		for j := 0; j < 2; j++ {
			activeHallRequests[i][j] = false
		}
	}

	for {
		select {
		case newHallReq := <-node.NewHallReqRx:
			fmt.Printf("Node %d received a new hall request: %v\n", node.ID, newHallReq)
			switch newHallReq.HallButton {

			case elevator.BT_HallUp:
				activeHallRequests[newHallReq.Floor][0] = true

			case elevator.BT_HallDown:
				activeHallRequests[newHallReq.Floor][1] = true

			case elevator.BT_Cab:
				fmt.Println("received a new hall requests, but the button type was invalid")
			}
			activeReq = true
			node.commandCh <- "getActiveElevStates"

		case newElevStates := <-node.ActiveElevStatesRx:
			fmt.Printf("Node %d received active elev states: %v\n", node.ID, newElevStates)
			if activeReq {
				HRAoutput := hallRequestAssigner.HRAalgorithm(newElevStates, activeHallRequests)
				for id, hallRequests := range *HRAoutput {
					nodeID, err := strconv.Atoi(id)
					if err != nil {
						fmt.Println("Error: ", err)
					}
					//sending hall requests to all nodes assuming all
					//nodes are connected nad not been disconnected after sending out internal states
					node.OutGoingHallAssignments <- messages.NewHallAssignments{NodeID: nodeID, HallAssignment: hallRequests, MessageID: 0}
				}
				activeReq = false
			}

		case connReq := <-node.ConnectionReqRx:
			// here, there may need to be some extra logic
			if connReq.TOLC.IsZero() {
				activeConnReq[connReq.NodeID] = connReq
				node.commandCh <- "getAllElevStates"
				activeConnectionReq = true
			}

		case allElevStates = <-node.AllElevStatesRx:
			if activeConnectionReq {
				//If activeConnectionReq is true, send all activeElevStates to nodes in activeConnReq
				for id := range activeConnReq {
					node.ElevStatesTx <- messages.ElevStates{NodeID: id, Direction: allElevStates[id].Direction, 
					Floor: allElevStates[id].Floor, CabRequest: allElevStates[id].CabRequest, 
					Behavior: allElevStates[id].Behavior}	
				}
				activeConnectionReq = false
			}

		case HA := <-node.HallAssignmentCompleteRx:
			// this logic could go somewhere else to clean up the master program
			if !recentHACompleteBuffer.Contains(HA.MessageID) {

				activeHallRequests[HA.Floor][HA.HallButton] = false
				node.AckTx <- messages.Ack{MessageID: HA.MessageID, NodeID: node.ID}

				recentHACompleteBuffer.Add(HA.MessageID)
			}
		
		case <-time.After(config.MASTER_TRANSMIT_INTERVAL):
			// Master transmitting global hall requests
			node.transmitEnableCh <- true
			node.GlobalHallRequestRx <- messages.GlobalHallRequest{HallRequests: activeHallRequests}

		}
	}
}
