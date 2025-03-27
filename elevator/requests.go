package elevator

import (
	"elev/config"
	"fmt"
)

type DirBehaviorPair struct {
	Dir      MotorDirection
	Behavior ElevatorBehavior
}

func RequestsAbove(e Elevator) bool {
	if e.Floor < 0 || e.Floor >= config.NUM_FLOORS {
		return false
	}
	for floor := e.Floor + 1; floor < config.NUM_FLOORS; floor++ {
		for btn := range config.NUM_BUTTONS {
			if e.Requests[floor][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(e Elevator) bool {
	if e.Floor < 0 || e.Floor >= config.NUM_FLOORS {
		return false
	}
	for floor := range e.Floor {
		for btn := range config.NUM_BUTTONS {
			if e.Requests[floor][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsHere(e Elevator) bool {
	if e.Floor < 0 || e.Floor >= config.NUM_FLOORS {
		return false
	}
	for btn := range config.NUM_BUTTONS {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func RequestsChooseDirection(e Elevator) DirBehaviorPair {
	switch e.Dir {
	case DirectionUp:
		if RequestsAbove(e) {
			return DirBehaviorPair{DirectionUp, Moving}
		} else if RequestsHere(e) {
			return DirBehaviorPair{DirectionDown, DoorOpen}
		} else if RequestsBelow(e) {
			return DirBehaviorPair{DirectionDown, Moving}
		} else {
			return DirBehaviorPair{DirectionStop, Idle}
		}
	case DirectionDown:
		if RequestsBelow(e) {
			return DirBehaviorPair{DirectionDown, Moving}
		} else if RequestsHere(e) {
			return DirBehaviorPair{DirectionUp, DoorOpen}
		} else if RequestsAbove(e) {
			return DirBehaviorPair{DirectionUp, Moving}
		} else {
			return DirBehaviorPair{DirectionStop, Idle}
		}
	case DirectionStop:
		if RequestsHere(e) {
			return DirBehaviorPair{DirectionStop, DoorOpen}
		} else if RequestsAbove(e) {
			return DirBehaviorPair{DirectionUp, Moving}
		} else if RequestsBelow(e) {
			return DirBehaviorPair{DirectionDown, Moving}
		} else {
			return DirBehaviorPair{DirectionStop, Idle}
		}
	default:
		return DirBehaviorPair{DirectionStop, Idle}
	}
}

func RequestsShouldStop(e Elevator) bool {
	switch e.Dir {
	case DirectionDown:
		return e.Requests[e.Floor][ButtonHallDown] || e.Requests[e.Floor][ButtonCab] || !RequestsBelow(e)
	case DirectionUp:
		return e.Requests[e.Floor][ButtonHallUp] || e.Requests[e.Floor][ButtonCab] || !RequestsAbove(e)
	case DirectionStop:
		fallthrough
	default:
		return true
	}
}

func RequestsShouldClearImmediately(e Elevator, btnFloor int, btnType ButtonType) bool {
	return e.Floor == btnFloor &&
		e.Behavior != StoppedBetweenFloors &&
		((e.Dir == DirectionUp && btnType == ButtonHallUp) ||
			(e.Dir == DirectionDown && btnType == ButtonHallDown) ||
			e.Dir == DirectionStop ||
			btnType == ButtonCab)
}

func RequestsClearAtCurrentFloor(e Elevator) Elevator {
	if e.Floor < 0 || e.Floor >= config.NUM_FLOORS {
		return e
	}

	e.Requests[e.Floor][ButtonCab] = false

	switch e.Dir {
	case DirectionUp:
		if e.Requests[e.Floor][ButtonHallUp] {
			e.Requests[e.Floor][ButtonHallUp] = false
			fmt.Printf("Clearing request at floor %d, button %d\n", e.Floor, ButtonHallUp)
		} else if e.Floor == config.NUM_FLOORS - 1 && e.Requests[e.Floor][ButtonHallDown] {
			e.Requests[e.Floor][ButtonHallDown] = false
			fmt.Printf("Clearing request at floor %d, button %d\n", e.Floor, ButtonHallDown)
		} else if !RequestsAbove(e) && !e.Requests[e.Floor][ButtonHallUp] && e.Requests[e.Floor][ButtonHallDown] {
			e.Requests[e.Floor][ButtonHallDown] = false
			fmt.Printf("Clearing request at floor %d, button %d\n", e.Floor, ButtonHallDown)
		}
	case DirectionDown:
		if e.Requests[e.Floor][ButtonHallDown] {
			e.Requests[e.Floor][ButtonHallDown] = false
			fmt.Printf("Clearing request at floor %d, button %d\n", e.Floor, ButtonHallDown)
		} else if e.Floor == 0 && e.Requests[e.Floor][ButtonHallUp] {
			e.Requests[e.Floor][ButtonHallUp] = false
			fmt.Printf("Clearing request at floor %d, button %d\n", e.Floor, ButtonHallUp)
		} else if !RequestsBelow(e) && !e.Requests[e.Floor][ButtonHallDown] && e.Requests[e.Floor][ButtonHallUp] {
			e.Requests[e.Floor][ButtonHallUp] = false
			fmt.Printf("Clearing request at floor %d, button %d\n", e.Floor, ButtonHallDown)
		}
	case DirectionStop:
		fallthrough
	default:
		if e.Requests[e.Floor][ButtonHallUp] {
			e.Requests[e.Floor][ButtonHallUp] = false
			fmt.Printf("Clearing request at floor %d, button %d\n", e.Floor, ButtonHallUp)
		} else if e.Requests[e.Floor][ButtonHallDown] {
			e.Requests[e.Floor][ButtonHallDown] = false
			fmt.Printf("Clearing request at floor %d, button %d\n", e.Floor, ButtonHallDown)
		}
	}
	return e
}
