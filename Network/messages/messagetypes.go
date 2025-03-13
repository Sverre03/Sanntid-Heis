package messages

import (
	"elev/elevator"
	"elev/util/config"
	"time"
)

// MessageType identifies the type of message being sent
type MessageType int

// Message types for elevator-to-node communication
const (
	MsgHallButtonEvent MessageType = iota
	MsgHallAssignmentComplete
	MsgElevatorState
	MsgDoorStuck
)

// ElevatorToNodeMsg encapsulates all messages sent from elevator to node
type ElevatorToNodeMsg struct {
	Type        MessageType
	ButtonEvent elevator.ButtonEvent   // For hall button events and completed hall assignments
	ElevState   elevator.ElevatorState // For elevator state updates
	IsDoorStuck bool                   // For door stuck status
}

// Message types for node-to-elevator communication
const (
	MsgHallAssignment MessageType = iota
	MsgRequestDoorState
)

// NodeToElevatorMsg encapsulates all messages sent from node to elevator
type NodeToElevatorMsg struct {
	Type            MessageType
	HallAssignments [config.NUM_FLOORS][2]bool // For assigning hall calls to the elevator
	CheckDoorState  bool                       // For checking the door state
}

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

// Message to update the state of the lights
type HallLightUpdate struct {
	LightStates       [config.NUM_FLOORS][2]bool
	MessageID         uint64
	ActiveElevatorIDs []int
}

// Message containing the states of your elevator, as well as your node id. This is broadcast as an alive message
type NodeElevState struct {
	NodeID    int
	ElevState elevator.ElevatorState
}

// Broadcast when you are in state disconnected. used to create a connection with other node
type ConnectionReq struct {
	TOLC      time.Time
	NodeID    int
	MessageID uint64
}

// Message from master to slaves on network, containing their new hall assignments
type NewHallAssignments struct {
	NodeID         int
	HallAssignment [config.NUM_FLOORS][2]bool
	MessageID      uint64
}

// When a slave gets a new hall button request, it broadcasts it to master in the form of a new hall request
type NewHallRequest struct {
	Floor      int
	HallButton elevator.ButtonType
}

// When a slave finishes an assigned hall order, it sends this message
type HallAssignmentComplete struct {
	Floor      int
	HallButton elevator.ButtonType
	MessageID  uint64
}
