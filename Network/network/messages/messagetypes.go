package messages

import "time"

const N_FLOORS int = 4

// a struct for acknowledging a message as received
type Ack struct {
	MessageID int
	NodeID    int
}

// event
type CabRequestINF struct {
	CabRequest     [N_FLOORS]bool
	MessageID      int
	ReceiverNodeID int
}

// information
type GlobalHallRequest struct {
	HallRequests [N_FLOORS][2]bool
}

// event
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

// event
type ConnectionReq struct {
	TOLC      time.Time
	MyNodeID  int
	MessageID int
}

// event
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

// event
type HallAssignmentComplete struct {
	Floor     int
	Dir       string
	MessageID int
}
