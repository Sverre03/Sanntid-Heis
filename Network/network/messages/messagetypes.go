package messages

import "time"

const N_FLOORS int = 4

// a struct for acknowledging a message as received
type Ack struct {
	MessageID  int
	NodeID int
}

type CabRequestINF struct{
	CabRequest [N_FLOORS]bool
	MessageID int
	ReceiverNodeID int
}

type GlobalHallRequest struct{
	HallRequests [N_FLOORS][2]bool
}

type HallLightUpdate struct {
	LightStates [N_FLOORS][2]bool
	MessageID int
}

type ElevStates struct{
	NodeID int
	Direction string
	Floor int
	CabRequest [N_FLOORS]bool
	Behavior string
}

type ConnectionReq struct {
	TOLC time.Time
	MyNodeID int
	MessageID int
}

type NewHallAssignments struct {
	NodeID int
	HallAssignment [N_FLOORS][2]bool
	MessageID int
}

type NewHallRequest struct {
	Floor int
	Dir string
}

type HallAssignmentComplete struct {
	Floor int
	Dir string
	MessageID int
}

