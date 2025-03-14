package elevator

import (
	"elev/util/config"
	"fmt"
)

type ElevatorBehavior int

const (
	Idle ElevatorBehavior = iota
	DoorOpen
	Moving
)

type Elevator struct {
	Floor           int
	Dir             MotorDirection
	Behavior        ElevatorBehavior
	Requests        [config.NUM_FLOORS][config.NUM_BUTTONS]bool
	IsObstructed    bool
	HallLightStates [config.NUM_FLOORS][config.NUM_BUTTONS - 1]bool // state of the hall lights
}

type ElevatorState struct {
	Behavior    ElevatorBehavior
	Floor       int
	Direction   MotorDirection
	CabRequests [config.NUM_FLOORS]bool
}

var ElevatorBehaviorToString = map[ElevatorBehavior]string{
	Idle:     "idle",
	DoorOpen: "doorOpen",
	Moving:   "moving",
}

var ElevatorDirectionToString = map[MotorDirection]string{
	DirectionUp:   "up",
	DirectionDown: "down",
	DirectionStop: "stop",
}

func GetCabRequestsAsElevState(elev Elevator) [config.NUM_FLOORS]bool {
	var cabRequests [config.NUM_FLOORS]bool
	for floor := 0; floor < config.NUM_FLOORS; floor++ {
		cabRequests[floor] = elev.Requests[floor][ButtonCab]
	}
	return cabRequests
}

func NewElevator() Elevator {
	return Elevator{
		Behavior: Idle,
		Floor:    -1,
		Dir:      DirectionStop,
		Requests: [config.NUM_FLOORS][config.NUM_BUTTONS]bool{},
	}
}

func PrintElevator(e Elevator) {
	behavior := ElevatorBehaviorToString[e.Behavior]
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
