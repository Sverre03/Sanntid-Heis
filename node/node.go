// Package node implements a fault-tolerant distributed system for elevator control.
// It provides mechanisms for master-slave coordination, connection between nodes,
// hall call distribution, and state synchronization between multiple elevator nodes.
package node

import (
	"elev/Network/messagehandler"
	"elev/Network/messages"
	"elev/Network/network/bcast"
	"elev/config"
	"elev/elevator"
	"elev/singleelevator"
	"fmt"
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
	TOLC               time.Time

	AckTx               chan messages.Ack                   // Send acks to udp broadcaster
	NodeElevStatesTx    chan messages.NodeElevState         // send your elev states to udp broadcaster
	NodeElevStateUpdate chan messagehandler.ElevStateUpdate // receive elevStateUpdate
	NewHallReqRx        chan messages.NewHallReq            // receive hall requests from udp receiver
	NewHallReqTx        chan messages.NewHallReq            // send hall requests to udp transmitter

	HallAssignmentTx  chan messages.NewHallAssignments // Sends hall assignments to hall assignment transmitter
	HallAssignmentsRx chan messages.NewHallAssignments // Receives hall assignments from udp receiver. Messages should be acked

	CabRequestInfoTx chan messages.CabRequestInfo // send known cab requests of another node to udp transmitter
	CabRequestInfoRx chan messages.CabRequestInfo // receive known cab requests from udp receiver

	GlobalHallRequestTx chan messages.GlobalHallRequest // update global hall request transmitter with the newest hall requests
	GlobalHallRequestRx chan messages.GlobalHallRequest // receive global hall requests from udp receiver

	ConnectionReqTx chan messages.ConnectionReq // send connection request messages to udp broadcaster
	ConnectionReqRx chan messages.ConnectionReq // receive connection request messages from udp receiver

	commandToServerTx chan string                      // Sends commands to the NodeElevStateServer (defined in Network/comm/receivers.go)
	NetworkEventRx    chan messagehandler.NetworkEvent // if no contact have been made within a timeout, "true" is sent on this channel

	// Elevator-Node communication
	ElevLightAndAssignmentUpdateTx chan singleelevator.LightAndAssignmentUpdate // channel for informing elevator of changes to hall button lights, hall assignments and cab assignments
	ElevatorEventRx                chan singleelevator.ElevatorEvent
	MyElevStatesRx                 chan elevator.ElevatorState

	// Channels for turning on and off the transmitter functions
	GlobalHallReqTransmitEnableTx       chan bool // channel that connects to GlobalHallRequestTransmitter, should be enabled when node is master
	HallRequestAssignerTransmitEnableTx chan bool // channel that connects to HallAssignmentsTransmitter, should be enabled when node is master
}

// initialize a network node and return a nodedata obj, needed for communication with the processes it starts
func MakeNode(id int, portNum string, bcastBroadcasterPort int, bcastReceiverPort int) *NodeData {

	node := &NodeData{
		ID:    id,
		State: Inactive,
		TOLC:  time.Time{},
	}

	node.AckTx = make(chan messages.Ack)
	ackRx := make(chan messages.Ack)

	node.NodeElevStatesTx = make(chan messages.NodeElevState)
	node.NodeElevStateUpdate = make(chan messagehandler.ElevStateUpdate)

	node.CabRequestInfoTx = make(chan messages.CabRequestInfo) //
	node.CabRequestInfoRx = make(chan messages.CabRequestInfo)

	node.ConnectionReqTx = make(chan messages.ConnectionReq)
	node.ConnectionReqRx = make(chan messages.ConnectionReq)

	HATransToBcastTx := make(chan messages.NewHallAssignments) // channel for communication from Hall Assignment Transmitter process to Broadcaster
	globalHallReqTransToBroadcast := make(chan messages.GlobalHallRequest)

	//Hall update
	node.NewHallReqRx = make(chan messages.NewHallReq)
	node.NewHallReqTx = make(chan messages.NewHallReq)
	// channels for enabling and disabling the transmitter functions
	node.GlobalHallReqTransmitEnableTx = make(chan bool)
	node.HallRequestAssignerTransmitEnableTx = make(chan bool)

	node.HallAssignmentTx = make(chan messages.NewHallAssignments)
	node.HallAssignmentsRx = make(chan messages.NewHallAssignments)

	node.GlobalHallRequestTx = make(chan messages.GlobalHallRequest) //
	node.GlobalHallRequestRx = make(chan messages.GlobalHallRequest)

	hallAssignmentsAckRx := make(chan messages.Ack)
	ConnectionReqAckRx := make(chan messages.Ack)

	node.ElevLightAndAssignmentUpdateTx = make(chan singleelevator.LightAndAssignmentUpdate, 3)
	node.ElevatorEventRx = make(chan singleelevator.ElevatorEvent)
	node.MyElevStatesRx = make(chan elevator.ElevatorState)

	node.commandToServerTx = make(chan string, 5)
	node.NetworkEventRx = make(chan messagehandler.NetworkEvent)

	node.GlobalHallReqTransmitEnableTx = make(chan bool)
	receiverToServerCh := make(chan messages.NodeElevState)

	// start process that broadcast all messages on these channels to udp
	go bcast.Broadcaster(bcastBroadcasterPort,
		node.AckTx,
		node.NodeElevStatesTx,
		HATransToBcastTx,
		node.CabRequestInfoTx,
		globalHallReqTransToBroadcast,
		node.ConnectionReqTx,
		node.NewHallReqTx)

	// start receiver process that listens for messages on the port
	go bcast.Receiver(bcastReceiverPort,
		ackRx,
		receiverToServerCh,
		node.HallAssignmentsRx,
		node.CabRequestInfoRx,
		node.GlobalHallRequestRx,
		node.ConnectionReqRx,
		node.NewHallReqRx)

	// process for distributing incoming acks in ackRx to different processes
	go messagehandler.IncomingAckDistributor(ackRx,
		hallAssignmentsAckRx,
		ConnectionReqAckRx)

	// process responsible for sending and making sure hall assignments are acknowledged
	go messagehandler.HallAssignmentsTransmitter(HATransToBcastTx,
		node.HallAssignmentTx,
		hallAssignmentsAckRx,
		node.HallRequestAssignerTransmitEnableTx)

	// the physical elevator program
	go singleelevator.ElevatorProgram(portNum,
		node.ElevatorEventRx,
		node.ElevLightAndAssignmentUpdateTx,
		node.MyElevStatesRx)

	// process that listens to active nodes on network
	go messagehandler.NodeElevStateServer(node.ID,
		node.commandToServerTx,
		node.NodeElevStateUpdate,
		receiverToServerCh,
		node.NetworkEventRx)

	// start the transmitter function
	go messagehandler.GlobalHallRequestsTransmitter(node.GlobalHallReqTransmitEnableTx,
		globalHallReqTransToBroadcast,
		node.GlobalHallRequestTx)

	return node
}

func sendCommandToServer(command string, node *NodeData) {
	select {
	case node.commandToServerTx <- command:
		// Command sent successfully
	default:
		// Command not sent, channel is full
		fmt.Printf("Warning: Command channel is full, command %s not sent\n", command)
	}
}
