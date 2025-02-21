package requests

import (
	"elev/elevator"
	"elev/elevio"
)

const NUM_FLOORS = 4
const N_BUTTONS = 3

type DirBehaviorPair struct {
	Dir      elevio.MotorDirection
	Behavior elevator.ElevatorBehavior
}

func RequestsAbove(e elevator.Elevator) bool {
	for floor := e.Floor + 1; floor < NUM_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.Requests[floor][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(e elevator.Elevator) bool {
	for floor := 0; floor < e.Floor; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.Requests[floor][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsHere(e elevator.Elevator) bool {
	for btn := 0; btn < N_BUTTONS; btn++ {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func RequestsChooseDirection(e elevator.Elevator) DirBehaviorPair {
	switch e.Dir {
	case elevio.MD_Up:
		if RequestsAbove(e) {
			return DirBehaviorPair{elevio.MD_Up, elevator.EB_Moving}
		} else if RequestsHere(e) {
			return DirBehaviorPair{elevio.MD_Down, elevator.EB_DoorOpen}
		} else if RequestsBelow(e) {
			return DirBehaviorPair{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return DirBehaviorPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Down:
		if RequestsBelow(e) {
			return DirBehaviorPair{elevio.MD_Down, elevator.EB_Moving}
		} else if RequestsHere(e) {
			return DirBehaviorPair{elevio.MD_Up, elevator.EB_DoorOpen}
		} else if RequestsAbove(e) {
			return DirBehaviorPair{elevio.MD_Up, elevator.EB_Moving}
		} else {
			return DirBehaviorPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Stop:
		if RequestsHere(e) {
			return DirBehaviorPair{elevio.MD_Stop, elevator.EB_DoorOpen}
		} else if RequestsAbove(e) {
			return DirBehaviorPair{elevio.MD_Up, elevator.EB_Moving}
		} else if RequestsBelow(e) {
			return DirBehaviorPair{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return DirBehaviorPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	default:
		return DirBehaviorPair{elevio.MD_Stop, elevator.EB_Idle}
	}
}

func RequestsShouldStop(e elevator.Elevator) bool {
	switch e.Dir {
	case elevio.MD_Down:
		return e.Requests[e.Floor][elevio.BT_HallDown] || e.Requests[e.Floor][elevio.BT_Cab] || !RequestsBelow(e)
	case elevio.MD_Up:
		return e.Requests[e.Floor][elevio.BT_HallUp] || e.Requests[e.Floor][elevio.BT_Cab] || !RequestsAbove(e)
	case elevio.MD_Stop:
		fallthrough
	default:
		return true
	}
}

func RequestsShouldClearImmediately(e elevator.Elevator, btnFloor int, btnType elevio.ButtonType) bool {
	return e.Floor == btnFloor && ((e.Dir == elevio.MD_Up && btnType == elevio.BT_HallUp) || (e.Dir == elevio.MD_Down && btnType == elevio.BT_HallDown) || e.Dir == elevio.MD_Stop || btnType == elevio.BT_Cab)
}

func RequestsClearAtCurrentFloor(e elevator.Elevator) elevator.Elevator {
	e.Requests[e.Floor][elevio.BT_Cab] = false
	switch e.Dir {
	case elevio.MD_Up:
		if !RequestsAbove(e) && !e.Requests[e.Floor][elevio.BT_HallUp] {
			e.Requests[e.Floor][elevio.BT_HallDown] = false
		}
		e.Requests[e.Floor][elevio.BT_HallUp] = false
	case elevio.MD_Down:
		if !RequestsBelow(e) && !e.Requests[e.Floor][elevio.BT_HallDown] {
			e.Requests[e.Floor][elevio.BT_HallUp] = false
		}
		e.Requests[e.Floor][elevio.BT_HallDown] = false
	case elevio.MD_Stop:
		fallthrough
	default:
		e.Requests[e.Floor][elevio.BT_HallUp] = false
		e.Requests[e.Floor][elevio.BT_HallDown] = false
	}
	return e
}
