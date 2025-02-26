package messages

import (
	"elev/util/config"
	"time"
)

// a struct for acknowledging the receival of a message
type Ack struct {
	MessageID int
	NodeID    int
}

// event - ack
type CabRequestINF struct {
	CabRequest     [config.NumFloors]bool
	MessageID      int
	ReceiverNodeID int
}

// information
type GlobalHallRequest struct {
	HallRequests [config.NumFloors][2]bool
}

// event - ack
type HallLightUpdate struct {
	LightStates       [config.NumFloors][2]bool
	MessageID         int
	ActiveElevatorIDs []int
}

// information
type ElevStates struct {
	NodeID     int
	Direction  string
	Floor      int
	CabRequest [config.NumFloors]bool
	Behavior   string
}

// event - ack
type ConnectionReq struct {
	TOLC      time.Time
	NodeID    int
	MessageID int
}

// event - ack
type NewHallAssignments struct {
	NodeID         int
	HallAssignment [config.NumFloors][2]bool
	MessageID      int
}

// event
type NewHallRequest struct {
	Floor int
	Dir   string
}

// event - ack
type HallAssignmentComplete struct {
	Floor     int
	Dir       string
	MessageID int
}
