package elevator

import (
	"elev/util/config"
	"fmt"
)

type ElevatorBehavior int

const (
	EB_Idle ElevatorBehavior = iota
	EB_DoorOpen
	EB_Moving
)

type Elevator struct {
	Floor        int
	Dir          MotorDirection
	Behavior     ElevatorBehavior
	Requests     [config.NUM_FLOORS][config.NUM_BUTTONS]bool
	IsObstructed bool
}

var ElevatorBehaviorToString = map[ElevatorBehavior]string{
	EB_Idle:     "idle",
	EB_DoorOpen: "doorOpen",
	EB_Moving:   "moving",
}

var ElevatorDirectionToString = map[MotorDirection]string{
	MD_Up:   "up",
	MD_Down: "down",
	MD_Stop: "stop",
}

func GetCabRequestsAsHRAElevState(elev Elevator) [config.NUM_FLOORS]bool {
	var cabRequests [config.NUM_FLOORS]bool
	for floor := 0; floor < config.NUM_FLOORS; floor++ {
		cabRequests[floor] = elev.Requests[floor][BT_Cab]
	}
	return cabRequests
}

func NewElevator() Elevator {
	return Elevator{
		Behavior: EB_Idle,
		Floor:    -1,
		Dir:      MD_Stop,
		Requests: [config.NUM_FLOORS][config.NUM_BUTTONS]bool{},
	}
}

func PrintElevator(e Elevator) {
	behavior := ElevatorBehaviorToString[e.Behavior]
	dir := "Stop"
	if e.Dir == MD_Up {
		dir = "Up"
	} else if e.Dir == MD_Down {
		dir = "Down"
	}
	fmt.Printf("Floor: %d\n", e.Floor)
	fmt.Printf("Direction: %s\n", dir)
	fmt.Printf("Behavior: %s\n", behavior)
	fmt.Printf("Obstructed: %t\n", e.IsObstructed)
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
