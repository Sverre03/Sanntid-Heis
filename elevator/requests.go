package elevator

import "elev/util/config"

type DirBehaviorPair struct {
	Dir      MotorDirection
	Behavior ElevatorBehavior
}

func RequestsAbove(e Elevator) bool {
	for floor := e.Floor + 1; floor < config.NUM_FLOORS; floor++ {
		for btn := 0; btn < config.NUM_BUTTONS; btn++ {
			if e.Requests[floor][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(e Elevator) bool {
	for floor := 0; floor < e.Floor; floor++ {
		for btn := 0; btn < config.NUM_BUTTONS; btn++ {
			if e.Requests[floor][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsHere(e Elevator) bool {
	for btn := 0; btn < config.NUM_BUTTONS; btn++ {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func RequestsChooseDirection(e Elevator) DirBehaviorPair {
	switch e.Dir {
	case MD_Up:
		if RequestsAbove(e) {
			return DirBehaviorPair{MD_Up, EB_Moving}
		} else if RequestsHere(e) {
			return DirBehaviorPair{MD_Down, EB_DoorOpen}
		} else if RequestsBelow(e) {
			return DirBehaviorPair{MD_Down, EB_Moving}
		} else {
			return DirBehaviorPair{MD_Stop, EB_Idle}
		}
	case MD_Down:
		if RequestsBelow(e) {
			return DirBehaviorPair{MD_Down, EB_Moving}
		} else if RequestsHere(e) {
			return DirBehaviorPair{MD_Up, EB_DoorOpen}
		} else if RequestsAbove(e) {
			return DirBehaviorPair{MD_Up, EB_Moving}
		} else {
			return DirBehaviorPair{MD_Stop, EB_Idle}
		}
	case MD_Stop:
		if RequestsHere(e) {
			return DirBehaviorPair{MD_Stop, EB_DoorOpen}
		} else if RequestsAbove(e) {
			return DirBehaviorPair{MD_Up, EB_Moving}
		} else if RequestsBelow(e) {
			return DirBehaviorPair{MD_Down, EB_Moving}
		} else {
			return DirBehaviorPair{MD_Stop, EB_Idle}
		}
	default:
		return DirBehaviorPair{MD_Stop, EB_Idle}
	}
}

func RequestsShouldStop(e Elevator) bool {
	switch e.Dir {
	case MD_Down:
		return e.Requests[e.Floor][BT_HallDown] || e.Requests[e.Floor][BT_Cab] || !RequestsBelow(e)
	case MD_Up:
		return e.Requests[e.Floor][BT_HallUp] || e.Requests[e.Floor][BT_Cab] || !RequestsAbove(e)
	case MD_Stop:
		fallthrough
	default:
		return true
	}
}

func RequestsShouldClearImmediately(e Elevator, btnFloor int, btnType ButtonType) bool {
	return e.Floor == btnFloor && ((e.Dir == MD_Up && btnType == BT_HallUp) || (e.Dir == MD_Down && btnType == BT_HallDown) || e.Dir == MD_Stop || btnType == BT_Cab)
}

func RequestsClearAtCurrentFloor(e Elevator) Elevator {
	e.Requests[e.Floor][BT_Cab] = false
	switch e.Dir {
	case MD_Up:
		if !RequestsAbove(e) && !e.Requests[e.Floor][BT_HallUp] {
			e.Requests[e.Floor][BT_HallDown] = false
		}
		e.Requests[e.Floor][BT_HallUp] = false
	case MD_Down:
		if !RequestsBelow(e) && !e.Requests[e.Floor][BT_HallDown] {
			e.Requests[e.Floor][BT_HallUp] = false
		}
		e.Requests[e.Floor][BT_HallDown] = false
	case MD_Stop:
		fallthrough
	default:
		e.Requests[e.Floor][BT_HallUp] = false
		e.Requests[e.Floor][BT_HallDown] = false
	}
	return e
}
