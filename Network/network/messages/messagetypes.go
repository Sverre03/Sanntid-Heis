package messages

import "time"

const N_FLOORS int = 4

// a struct for acknowledging the receival of a message
type Ack struct {
	MessageID int
	NodeID    int
}

// event - ack
type CabRequestINF struct {
	CabRequest     [N_FLOORS]bool
	MessageID      int
	ReceiverNodeID int
}

// information
type GlobalHallRequest struct {
	HallRequests [N_FLOORS][2]bool
}

// event - ack
type HallLightUpdate struct {
	LightStates [N_FLOORS][2]bool
	MessageID   int
}

// information
type ElevStates struct {
	NodeID     int
	Direction  string
	Floor      int
	CabRequest [N_FLOORS]bool
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
	HallAssignment [N_FLOORS][2]bool
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
