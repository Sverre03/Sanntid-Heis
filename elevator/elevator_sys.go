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

type ElevatorState struct {
	Floor             int
	Direction         MotorDirection
	Behavior          ElevatorBehavior
	CabRequests       [config.NUM_FLOORS]bool
	MyHallAssignments [config.NUM_FLOORS][2]bool
	NodeID            int
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
	default:
		return fmt.Sprintf("unknown(%d)", int(eb))
	}
}

func (button ButtonType) String() string {
	switch button {
	case ButtonHallUp:
		return "HallUp"
	case ButtonHallDown:
		return "HallDown"
	case ButtonCab:
		return "Cab"
	default:
		return "Unknown"
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

func GetCabRequestsAsElevState(elev Elevator) [config.NUM_FLOORS]bool {
	var cabRequests [config.NUM_FLOORS]bool
	for floor := range config.NUM_FLOORS {
		cabRequests[floor] = elev.Requests[floor][ButtonCab]
	}
	return cabRequests
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

func PrintElevator(e Elevator) {
	behavior := e.Behavior.String()
	dir := "Stop"
	if e.Dir == DirectionUp {
		dir = "Up"
	} else if e.Dir == DirectionDown {
		dir = "Down"
	}
	fmt.Printf("Floor: %d\n", e.Floor)
	fmt.Printf("Direction: %s\n", dir)
	fmt.Printf("Behavior: %s\n", behavior)
	fmt.Printf("Obstructed: %t\n", e.IsObstructed)
	fmt.Println("Request Matrix:")
	for floor := len(e.Requests) - 1; floor >= 0; floor-- {
		fmt.Printf("Floor %d: ", floor)
		for btn := range len(e.Requests[floor]) {
			if e.Requests[floor][btn] {
				fmt.Print("# ")
			} else {
				fmt.Print("- ")
			}
		}
		fmt.Println()
	}
}
