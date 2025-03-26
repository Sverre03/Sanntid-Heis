package messages

import (
	"elev/config"
	"elev/elevator"
	"time"
)

// MessageType identifies the type of message being sent
type MessageType int

// Message types for elevator-to-node communication
const (
	MsgHallAssignment   MessageType = iota // Transmits assigned hall calls to elevator, [floor][up/down]
	MsgRequestDoorState                    // Sends a request to the elevator to check its door state
)

// a struct for acknowledging a message is received
type Ack struct {
	MessageID uint64 // the id of the message you received
	NodeID    int
}

// Message that contains the cab requests of a single elevator, sent from master to a disconnected node on reconnect as a backup of your internal states
type CabRequestInfo struct {
	CabRequest     [config.NUM_FLOORS]bool
	ReceiverNodeID int
}

// Message with the hall requests of the system. Meant to be broadcast by master and only master at a fixed interval. If you receive this message, it means a master exists
type GlobalHallRequest struct {
	HallRequests [config.NUM_FLOORS][2]bool
}

// Message containing the states of your elevator, as well as your node id. This is broadcast as an alive message
type NodeElevState struct {
	NodeID    int
	ElevState elevator.ElevatorState
}

// Broadcast when you are in state disconnected. used to create a connection with other node
type ConnectionReq struct {
	TOLC   time.Time
	NodeID int
}

// Message from master to slaves on network, containing their new hall assignments
type NewHallAssignments struct {
	NodeID                int
	HallAssignment        [config.NUM_FLOORS][2]bool
	MessageID             uint64 // message identifier, generated in the transmitter
	HallAssignmentCounter int
}

type NewHallReq struct {
	NodeID  int
	HallReq elevator.ButtonEvent
}
