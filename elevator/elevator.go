package elevator

import (
	"Driver-go/elevio"
	"fmt"
)

const _numFloors = 4
const _numButtons = 3

type ElevatorBehavior int

const (
	EB_Idle ElevatorBehavior = iota
	EB_DoorOpen
	EB_Moving
)

type Elevator struct {
	Floor    int
	Dir      elevio.MotorDirection
	Behavior ElevatorBehavior
	Requests [_numFloors][_numButtons]bool
	ID       string
}

var ElevatorBehaviorToString = map[ElevatorBehavior]string{
	EB_Idle:     "Idle",
	EB_DoorOpen: "DoorOpen",
	EB_Moving:   "Moving",
}

func NewElevator(ID string) Elevator {
	return Elevator{
		Behavior: EB_Idle,
		Floor:    -1,
		Dir:      elevio.MD_Stop,
		Requests: [_numFloors][_numButtons]bool{},
		ID:       ID,
	}
}

func PrintElevator(e Elevator) {
	behavior := ElevatorBehaviorToString[e.Behavior]
	dir := "Stop"
	if e.Dir == elevio.MD_Up {
		dir = "Up"
	} else if e.Dir == elevio.MD_Down {
		dir = "Down"
	}
	fmt.Printf("Elevator ID: %s\n", e.ID)
	fmt.Printf("Floor: %d\n", e.Floor)
	fmt.Printf("Direction: %s\n", dir)
	fmt.Printf("Behavior: %s\n", behavior)
	fmt.Println("Request Matrix:")
	for floor := len(e.Requests) - 1; floor >= 0; floor-- {
		fmt.Printf("Floor %d: ", floor)
		for btn := 0; btn < len(e.Requests[floor]); btn++ {
			if e.Requests[floor][btn] {
				fmt.Print("# ")
			} else {
				fmt.Print("- ")
			}
		}
		fmt.Println()
	}
}
