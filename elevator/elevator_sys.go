package elevator

import (
	"elev/config"
	"fmt"
)

type ElevatorBehavior int

const (
	Idle ElevatorBehavior = iota
	DoorOpen
	Moving
	StoppedBetweenFloors
)

type Elevator struct {
	Floor                int
	Dir                  MotorDirection
	Behavior             ElevatorBehavior
	Requests             [config.NUM_FLOORS][config.NUM_BUTTONS]bool
	HallLightStates      [config.NUM_FLOORS][config.NUM_BUTTONS - 1]bool
	IsObstructed         bool
	DoorStuckTimerActive bool
}

type ElevatorStateReport struct {
	Floor             int
	Direction         MotorDirection
	Behavior          ElevatorBehavior
	CabRequests       [config.NUM_FLOORS]bool
	MyHallAssignments [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool
	HACounterVersion  int
}

// String returns a string representation of the ElevatorBehavior
func (eb ElevatorBehavior) String() string {
	switch eb {
	case Idle:
		return "idle"
	case DoorOpen:
		return "doorOpen"
	case Moving:
		return "moving"
	case StoppedBetweenFloors:
		return "stoppedBetweenFloors"
	default:
		return fmt.Sprintf("unknown(%d)", int(eb))
	}
}

// String returns a string representation of the MotorDirection
func (md MotorDirection) String() string {
	switch md {
	case DirectionUp:
		return "up"
	case DirectionDown:
		return "down"
	case DirectionStop:
		return "stop"
	default:
		return fmt.Sprintf("unknown(%d)", int(md))
	}
}

func NewElevator() Elevator {
	return Elevator{
		Behavior:             Idle,
		Floor:                -1,
		Dir:                  DirectionStop,
		Requests:             [config.NUM_FLOORS][config.NUM_BUTTONS]bool{},
		DoorStuckTimerActive: false,
	}
}
