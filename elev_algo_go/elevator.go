package elevator

import (
	"Driver-go/elevio"
	"fmt"
)

type ElevatorBehaviour int

const (
	EB_Idle ElevatorBehaviour = iota
	EB_DoorOpen
	EB_Moving
)

type Elevator struct {
	Floor         int
	Dir           elevio.MotorDirection
	Behaviour     ElevatorBehaviour
	RequestMatrix [elevio.NumFloors][elevio.NumButtons]bool
	ID            string
}

var ElevatorBehaviourToString = map[ElevatorBehaviour]string{
	EB_Idle:     "Idle",
	EB_DoorOpen: "DoorOpen",
	EB_Moving:   "Moving",
}

func NewElevator(ID string) Elevator {
	return Elevator{
		Floor:     -1,
		Dir:       elevio.MD_Stop,
		Behaviour: EB_Idle,
		ID:        ID,
	}
}

func PrintElevator(e Elevator) {
	behaviour := ElevatorBehaviourToString[e.Behaviour]
	dir := "Stop"
	if e.Dir == elevio.MD_Up {
		dir = "Up"
	} else if e.Dir == elevio.MD_Down {
		dir = "Down"
	}
	fmt.Printf("Elevator ID: %s\n", e.ID)
	fmt.Printf("Floor: %d\n", e.Floor)
	fmt.Printf("Direction: %s\n", dir)
	fmt.Printf("Behaviour: %s\n", behaviour)
	fmt.Println("Request Matrix:")
	for floor := len(e.RequestMatrix) - 1; floor >= 0; floor-- {
		fmt.Printf("Floor %d: ", floor)
		for btn := 0; btn < len(e.RequestMatrix[floor]); btn++ {
			if e.RequestMatrix[floor][btn] {
				fmt.Print("# ")
			} else {
				fmt.Print("- ")
			}
		}
		fmt.Println()
	}
}
