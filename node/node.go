// Package node implements a fault-tolerant distributed system for elevator control.
// It provides mechanisms for master-slave coordination, connection between nodes,
// hall call distribution, and state synchronization between multiple elevator nodes.
package node

import (
	"context"
	"elev/Network/comm"
	"elev/Network/network/bcast"
	"elev/Network/network/messages"
	"elev/elevator"
	"elev/elevatoralgo"
	"elev/util/config"
	"fmt"
	"time"

	"github.com/looplab/fsm"
)

// NodeData represents a node in the distributed elevator system.
// It contains the node's state machine, communication channels,
// and necessary data for the node to function.
type NodeData struct {
	ID        int
	NodeState *fsm.FSM

	GlobalHallRequests [config.NUM_FLOORS][2]bool

	AckTx        chan messages.Ack
	ElevStatesTx chan messages.ElevStates

	HallAssignmentTx  chan messages.NewHallAssignments // Transmits hall assignments to elevators on the network
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

	commandTx          chan string                      // Sends commands to the ElevStateListener (defined in Network/comm/receivers.go)
	ActiveElevStatesRx chan map[int]messages.ElevStates // Receives the state of the other active node's elevators
	AllElevStatesRx    chan map[int]messages.ElevStates
	TOLCRx             chan time.Time // Receives the Time of Last Contact
	ActiveNodeIDsRx    chan []int     // Receives the IDs of the active nodes on the network

	NewHallReqTx chan messages.NewHallRequest // Sends new hall requests to other nodes
	NewHallReqRx chan messages.NewHallRequest // Receives new hall requests from other nodes

	// Elevator-Node communication channels
	ElevatorHallButtonEventTx chan elevator.ButtonEvent   // Transmit assigned hall calls to elevator
	ElevatorHallButtonEventRx chan elevator.ButtonEvent   // Receives local hall button presses from node
	ElevatorHRAStatesRx       chan elevator.ElevatorState // Receives the elevator's internal state
	IsDoorStuckCh             chan bool                   // Receives the elevator's door state (if it is stuck or not)
	RequestDoorStateCh        chan bool                   // Sends a request to the elevator to check its door state

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
	node.ElevatorHRAStatesRx = make(chan elevator.ElevatorState)
	node.IsDoorStuckCh = make(chan bool)
	node.RequestDoorStateCh = make(chan bool)
	go elevatoralgo.ElevatorProgram(node.ElevatorHallButtonEventRx, node.ElevatorHRAStatesRx, node.ElevatorHallButtonEventTx, node.IsDoorStuckCh, node.RequestDoorStateCh)

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
