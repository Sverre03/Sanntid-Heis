package node

import (
	"Network/network/bcast"
	"Network/network/messages"
	"context"
	"fmt"
	"time"

	"github.com/looplab/fsm"
)

type NodeData struct {
	ID        int
	NodeState *fsm.FSM

	TOLC                      time.Time
	Elevator                  *Elevator
	TaskQueue                 []string
	GlobalHallRequests        []string
	LastKnownStatesOfAllNodes map[int]string

	AckTx chan messages.Ack
	AckRx chan messages.Ack

	ElevStatesTx chan messages.ElevStates
	ElevStatesRx chan messages.ElevStates

	HallAssignmentsTx chan messages.NewHallAssignments
	HallAssignmentsRx chan messages.NewHallAssignments

	CabRequestInfoTx chan messages.CabRequestINF
	CabRequestInfoRx chan messages.CabRequestINF

	GlobalHallRequestTx chan messages.GlobalHallRequest
	GlobalHallRequestRx chan messages.GlobalHallRequest

	HallLightUpdateTx chan messages.HallLightUpdate
	HallLightUpdateRx chan messages.HallLightUpdate

	ConnectionReqTx chan messages.ConnectionReq
	ConnectionReqRx chan messages.ConnectionReq

	NewHallReqTx chan messages.NewHallRequest
	NewHallReqRx chan messages.NewHallRequest

	HallAssignmentCompleteTx chan messages.HallAssignmentComplete
	HallAssignmentCompleteRx chan messages.HallAssignmentComplete
}

func Node(id int) *NodeData {
	node := &NodeData{
		ID:                        id,
		TOLC:                      time.Time{},
		Elevator:                  &Elevator{},
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
	node.AckRx = make(chan messages.Ack)

	node.ElevStatesTx = make(chan messages.ElevStates)
	node.ElevStatesRx = make(chan messages.ElevStates)

	node.HallAssignmentsTx = make(chan messages.NewHallAssignments)
	node.HallAssignmentsRx = make(chan messages.NewHallAssignments)

	node.CabRequestInfoTx = make(chan messages.CabRequestINF)
	node.CabRequestInfoRx = make(chan messages.CabRequestINF)

	node.GlobalHallRequestTx = make(chan messages.GlobalHallRequest)
	node.GlobalHallRequestRx = make(chan messages.GlobalHallRequest)

	node.HallLightUpdateTx = make(chan messages.HallLightUpdate)
	node.HallLightUpdateRx = make(chan messages.HallLightUpdate)

	node.ConnectionReqTx = make(chan messages.ConnectionReq)
	node.ConnectionReqRx = make(chan messages.ConnectionReq)

	node.NewHallReqTx = make(chan messages.NewHallRequest)
	node.NewHallReqRx = make(chan messages.NewHallRequest)

	node.HallAssignmentCompleteTx = make(chan messages.HallAssignmentComplete)
	node.HallAssignmentCompleteRx = make(chan messages.HallAssignmentComplete)

	go bcast.Transmitter(PortNum, node.AckTx, node.ElevStatesTx, node.HallAssignmentsTx, node.CabRequestInfoTx, node.GlobalHallRequestTx, node.HallLightUpdateTx, node.ConnectionReqTx, node.NewHallReqTx, node.HallAssignmentCompleteTx)
	go bcast.Receiver(PortNum, node.AckRx, node.ElevStatesRx, node.HallAssignmentsRx, node.CabRequestInfoRx, node.GlobalHallRequestRx, node.HallLightUpdateRx, node.ConnectionReqRx, node.NewHallReqRx, node.HallAssignmentCompleteRx)

	return node
}

func (node *NodeData) onEnterInactive(_ context.Context, e *fsm.Event) {
	fmt.Printf("Node %d er nå INACTIVE. Med TOLC lik %s \node", node.ID, node.TOLC)
	InactiveProgram(node)
}

func (node *NodeData) onEnterDisconnected(_ context.Context, e *fsm.Event) {
	node.TOLC = time.Time{}
	fmt.Printf("Node %d er nå DISCONNECTED. Med TOLC lik %s \node", node.ID, node.TOLC)
	DisconnectedProgram()
}

func (node *NodeData) onEnterSlave(_ context.Context, e *fsm.Event) {
	node.TOLC = time.Now()
	fmt.Printf("Node %d er nå SLAVE. Med TOLC lik %s \node", node.ID, node.TOLC)
	SlaveProgram()
}

func (node *NodeData) onEnterMaster(_ context.Context, e *fsm.Event) {
	node.TOLC = time.Now()
	fmt.Printf("Node %d er nå MASTER. Med TOLC lik %s \node", node.ID, node.TOLC)
	MasterProgram()
}

func InactiveProgram(node *NodeData) {
	if err := node.NodeState.Event(context.Background(), "initialize"); err != nil {
		fmt.Println("Error:", err)
	}
}

func DisconnectedProgram(node *NodeData) {

	for {
		select {
		case conReq := <-node.ConnectionReqRx:
			if node.ID != conReq.MyNodeID {
				node.AckTx <- messages.Ack{MessageID: conReq.MessageID, NodeID: node.ID}
			}
		}
	}
}
