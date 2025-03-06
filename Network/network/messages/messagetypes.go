package messages

import (
	"elev/elevator"
	"elev/util/config"
	"time"
)

// a struct for acknowledging the receival of a message
type Ack struct {
	MessageID uint64
	NodeID    int
}

// event - ack
type CabRequestInfo struct {
	CabRequest     [config.NUM_FLOORS]bool
	ReceiverNodeID int
}

// information
type GlobalHallRequest struct {
	HallRequests [config.NUM_FLOORS][2]bool
}

// event - ack
type HallLightUpdate struct {
	LightStates       [config.NUM_FLOORS][2]bool
	MessageID         uint64
	ActiveElevatorIDs []int
}

// information
type NodeElevState struct {
	NodeID    int
	ElevState elevator.ElevatorState
}

// event - ack
type ConnectionReq struct {
	TOLC      time.Time
	NodeID    int
	MessageID uint64
}

// event - ack
type NewHallAssignments struct {
	NodeID         int
	HallAssignment [config.NUM_FLOORS][2]bool
	MessageID      uint64
}

// event
type NewHallRequest struct {
	Floor      int
	HallButton elevator.ButtonType
}

// event - ack
type HallAssignmentComplete struct {
	Floor      int
	HallButton elevator.ButtonType
	MessageID  uint64
}
