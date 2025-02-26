package node

import (
	"context"
	"elev/Network/comm"
	"elev/Network/network/bcast"
	"elev/Network/network/messages"
	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"elev/util/config"
	"fmt"
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

	commandCh chan string

	HallAssignmentAckRx  chan messages.Ack
	HallLightUpdateAckRx chan messages.Ack
	ConnectionReqAckRx   chan messages.Ack

	NewHallReqTx chan messages.NewHallRequest
	NewHallReqRx chan messages.NewHallRequest

	HallAssignmentCompleteRx chan messages.HallAssignmentComplete
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
	timeOfLastContactCh := make(chan time.Time)
	elevStatesCh := make(chan map[int]messages.ElevStates)
	activeNodeIDsC := make (chan []int)
	elevStatesRx := make(chan messages.ElevStates)


	go bcast.Transmitter(config.PORT_NUM, node.AckTx, node.ElevStatesTx, HallAssignmentsTx, CabRequestInfoTx, GlobalHallRequestTx, HallLightUpdateTx, node.ConnectionReqTx, node.NewHallReqTx, HallAssignmentCompleteTx)
	go bcast.Receiver(config.PORT_NUM, AckRx, ElevStatesRx, node.HallAssignmentsRx, node.CabRequestInfoRx, node.GlobalHallRequestRx, node.HallLightUpdateRx, node.ConnectionReqRx, node.NewHallReqRx, node.HallAssignmentCompleteRx)
	go comm.HallAssignmentsTransmitter(HallAssignmentsTx, node.OutGoingHallAssignments, HallAssignmentsAckTx)
	go comm.ElevStatesListener(node.commandCh, timeOfLastContactCh, elevStatesCh, activeNodeIDsC, elevStatesRx)
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
	timeOfLastContact := time.Time{} // placeholder for getting from server
	msgID := 0                       // placeholder for using "getmessageid function"

	myConnReq := messages.ConnectionReq{TOLC: timeOfLastContact, NodeID: node.ID, MessageID: msgID}
	incomingConnRequests := make(map[int]messages.ConnectionReq)

	// ID of the node we currently are trying to connect with
	currentFriendID := 0

	// isConnRequestActive := false

	for {
		select {

		case <-node.GlobalHallRequestRx:
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
	fmt.Printf("Node %d er nå MASTER. Med TOLC lik %s \node", node.ID, node.TOLC)
}

func MasterProgram(node *NodeData) {

	for {
		var hallRequests [][2]bool                           //placeholder for getting from server
		var allElevStates []hallRequestAssigner.HRAElevState //placeholder for getting from server

		messageID, err := comm.GenerateMessageID(comm.NEW_HALL_ASSIGNMENT)
		if err != nil {
			fmt.Printf("Fatal error, invalid message id used")
		}

		inputFormat := hallRequestAssigner.InputFunction(allElevStates, hallRequests)
		outputFormat := hallRequestAssigner.OutputFunction(inputFormat)

		node.OutGoingHallAssignments <- messages.NewHallAssignments{node.ID, outputFormat, messageID}
	}
}
