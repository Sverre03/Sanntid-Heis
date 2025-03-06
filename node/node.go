// Package node implements a fault-tolerant distributed system for elevator control.
// It provides mechanisms for master-slave coordination, connection between nodes,
// hall call distribution, and state synchronization between multiple elevator nodes.
package node

import (
	"elev/Network/comm"
	"elev/Network/network/bcast"
	"elev/Network/network/messages"
	"elev/elevator"
	"elev/elevatoralgo"
	"elev/util/config"
	"time"
)

// NodeData represents a node in the distributed elevator system.
// It contains the node's state machine, communication channels,
// and necessary data for the node to function.

type nodestate int

const (
	Inactive nodestate = iota
	Disconnected
	Master
	Slave
)

type NodeData struct {
	ID                 int
	State              nodestate
	GlobalHallRequests [config.NUM_FLOORS][2]bool

	AckTx        chan messages.Ack
	ElevStatesTx chan messages.NodeElevState

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

	commandToServerTx            chan string                         // Sends commands to the ElevStateListener (defined in Network/comm/receivers.go)
	ActiveElevStatesFromServerRx chan map[int]messages.NodeElevState // Receives the state of the other active node's elevators
	AllElevStatesFromServerRx    chan map[int]messages.NodeElevState
	TOLCFromServerRx             chan time.Time // Receives the Time of Last Contact
	ActiveNodeIDsFromServerRx    chan []int     // Receives the IDs of the active nodes on the network

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

func CreateNode(id int) *NodeData {

	node := &NodeData{
		ID:    id,
		State: Inactive,
	}

	// broadcast channels
	node.AckTx = make(chan messages.Ack)
	node.ElevStatesTx = make(chan messages.NodeElevState)
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
	elevStatesRx := make(chan messages.NodeElevState)

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

	node.commandToServerTx = make(chan string)
	node.TOLCFromServerRx = make(chan time.Time)
	node.ActiveElevStatesFromServerRx = make(chan map[int]messages.NodeElevState)
	node.AllElevStatesFromServerRx = make(chan map[int]messages.NodeElevState)
	node.ActiveNodeIDsFromServerRx = make(chan []int)
	go comm.NodeElevStateServer(node.ID, node.commandToServerTx, node.TOLCFromServerRx, node.ActiveElevStatesFromServerRx, node.ActiveNodeIDsFromServerRx, elevStatesRx, node.AllElevStatesFromServerRx)

	node.GlobalHallRequestTx = make(chan messages.GlobalHallRequest) //
	node.GlobalHallReqTransmitEnableTx = make(chan bool)
	go comm.GlobalHallRequestsTransmitter(node.GlobalHallReqTransmitEnableTx, globalHallReqTransToBroadcast, node.GlobalHallRequestTx)

	node.HallLightUpdateTx = make(chan messages.HallLightUpdate) //
	go comm.LightUpdateTransmitter(lightUpdateTransToBroadcast, node.HallLightUpdateTx, lightUpdateAckRx)

	return node
}
