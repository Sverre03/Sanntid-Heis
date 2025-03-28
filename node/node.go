// Package node implements a fault-tolerant distributed system for elevator control.
// It provides mechanisms for master-slave coordination, connection between nodes,
// hall call distribution, and state synchronization between multiple elevator nodes.
package node

import (
	"elev/config"
	"elev/elevator"
	"elev/network/bcast"
	"elev/network/communication"
	"elev/network/messages"
	"elev/singleelevator"
)

// NodeData represents a node in the distributed elevator system.
// It contains the node's state, communication channels and necessary data for the node to function.

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
	GlobalHallRequests [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool
	ContactCounter     uint64 // counter that is set at contact with master

	AckTx                      chan messages.Ack                  // Send acks to udp broadcaster
	NodeElevStatesTx           chan messages.NodeElevState        // send your elev states to udp broadcaster
	ElevStateUpdatesFromServer chan communication.ElevStateUpdate // receive elevStateUpdates from StateMonitorServer
	NewHallReqRx               chan messages.NewHallReq           // receive hall requests from udp receiver
	NewHallReqTx               chan messages.NewHallReq           // send hall requests to udp transmitter

	HallAssignmentTx  chan messages.NewHallAssignments // Sends hall assignments to hall assignment transmitter
	HallAssignmentsRx chan messages.NewHallAssignments // Receives hall assignments from udp receiver. Messages should be acked

	CabRequestInfoTx chan messages.CabRequestInfo // send known cab requests of another node to udp transmitter
	CabRequestInfoRx chan messages.CabRequestInfo // receive known cab requests from udp receiver

	GlobalHallRequestTx chan messages.GlobalHallRequest // update global hall request transmitter with the newest hall requests
	GlobalHallRequestRx chan messages.GlobalHallRequest // receive global hall requests from udp receiver

	ConnectionReqTx chan messages.ConnectionReq // send connection request messages to udp broadcaster
	ConnectionReqRx chan messages.ConnectionReq // receive connection request messages from udp receiver

	commandToServerTx chan string                     // Sends commands to the NodeElevStateServer (defined in Network/comm/receivers.go)
	NetworkEventRx    chan communication.NetworkEvent // if no contact have been made within a timeout, "true" is sent on this channel

	// Elevator-Node communication
	ElevLightAndAssignmentUpdateTx chan singleelevator.LightAndAssignmentUpdate // channel for informing elevator of changes to hall button lights, hall assignments and cab assignments
	ElevatorEventRx                chan singleelevator.ElevatorEvent
	MyElevStatesRx                 chan elevator.ElevatorStateReport

	HallAssignmentTransmitterEnableTx chan bool // channel that connects to HallAssignmentsTransmitter, should be enabled when node is master
}

// initialize a network node and return a nodedata obj, needed for communication with the processes it starts
func MakeNode(id int, portNum string, bcastPort int) *NodeData {

	node := &NodeData{
		ID:             id,
		State:          Inactive,
		ContactCounter: 0,
	}

	node.AckTx = make(chan messages.Ack)

	node.NodeElevStatesTx = make(chan messages.NodeElevState)
	node.ElevStateUpdatesFromServer = make(chan communication.ElevStateUpdate)

	node.CabRequestInfoTx = make(chan messages.CabRequestInfo)
	node.CabRequestInfoRx = make(chan messages.CabRequestInfo)

	node.ConnectionReqTx = make(chan messages.ConnectionReq)
	node.ConnectionReqRx = make(chan messages.ConnectionReq)

	HAssignmentTransmitterToBcastTx := make(chan messages.NewHallAssignments) // channel for communication from Hall Assignment Transmitter process to Broadcaster

	node.NewHallReqRx = make(chan messages.NewHallReq)
	node.NewHallReqTx = make(chan messages.NewHallReq)

	node.HallAssignmentTransmitterEnableTx = make(chan bool)

	node.HallAssignmentTx = make(chan messages.NewHallAssignments)
	node.HallAssignmentsRx = make(chan messages.NewHallAssignments)

	node.GlobalHallRequestTx = make(chan messages.GlobalHallRequest) //
	node.GlobalHallRequestRx = make(chan messages.GlobalHallRequest)

	hallAssignmentsAckRx := make(chan messages.Ack)

	node.ElevLightAndAssignmentUpdateTx = make(chan singleelevator.LightAndAssignmentUpdate, 5)
	node.ElevatorEventRx = make(chan singleelevator.ElevatorEvent)
	node.MyElevStatesRx = make(chan elevator.ElevatorStateReport)

	node.commandToServerTx = make(chan string, 5)
	node.NetworkEventRx = make(chan communication.NetworkEvent, 5)

	receiverToServerCh := make(chan messages.NodeElevState)

	// start process that broadcast all messages on these channels to udp
	go bcast.Broadcaster(bcastPort,
		node.AckTx,
		node.NodeElevStatesTx,
		HAssignmentTransmitterToBcastTx,
		node.CabRequestInfoTx,
		node.GlobalHallRequestTx,
		node.ConnectionReqTx,
		node.NewHallReqTx)

	// start receiver process that listens for messages on the port
	go bcast.Receiver(
		bcastPort,
		hallAssignmentsAckRx,
		receiverToServerCh,
		node.HallAssignmentsRx,
		node.CabRequestInfoRx,
		node.GlobalHallRequestRx,
		node.ConnectionReqRx,
		node.NewHallReqRx)

	// process responsible for sending and making sure hall assignments are acknowledged
	go communication.HallAssignmentsTransmitter(
		HAssignmentTransmitterToBcastTx,
		node.HallAssignmentTx,
		hallAssignmentsAckRx,
		node.HallAssignmentTransmitterEnableTx)

	// the physical elevator program
	go singleelevator.ElevatorProgram(portNum,
		node.ElevLightAndAssignmentUpdateTx,
		node.ElevatorEventRx,
		node.MyElevStatesRx)

	// process that listens to active nodes on network
	go communication.ConnectionMonitorServer(node.ID,
		node.commandToServerTx,
		node.ElevStateUpdatesFromServer,
		receiverToServerCh,
		node.NetworkEventRx)

	return node
}

// functions used in the state machines of the different nodes
func makeHallAssignmentAndLightMessage(
	hallAssignments [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool,
	globalHallReq [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool,
	hallAssignmentCounterValue int) singleelevator.LightAndAssignmentUpdate {
	var newMessage singleelevator.LightAndAssignmentUpdate
	newMessage.HallAssignments = hallAssignments
	newMessage.LightStates = globalHallReq
	newMessage.OrderType = singleelevator.HallAssignment
	newMessage.HallAssignmentCounterValue = hallAssignmentCounterValue
	return newMessage
}

func makeLightMessage(hallReq [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool) singleelevator.LightAndAssignmentUpdate {
	var newMessage singleelevator.LightAndAssignmentUpdate
	newMessage.LightStates = hallReq
	newMessage.OrderType = singleelevator.LightUpdate
	return newMessage
}

func makeNewHallReq(nodeID int, elevMsg singleelevator.ElevatorEvent) messages.NewHallReq {
	return messages.NewHallReq{
		NodeID: nodeID,
		HallReq: elevator.ButtonEvent{
			Floor:  elevMsg.ButtonEvent.Floor,
			Button: elevMsg.ButtonEvent.Button,
		},
	}
}
