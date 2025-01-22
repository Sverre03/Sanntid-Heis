func RequestsAbove(e Elevator) bool {
    for f := e.floor + 1; f < N_FLOORS; f++ {
        for btn := 0; btn < N_BUTTONS; btn++ {
            if e.requests[f][btn] {
                return true
            }
        }
    }
    return false
}

func RequestsBelow(e Elevator) bool {
    for f := 0; f < e.floor; f++ {
        for btn := 0; btn < N_BUTTONS; btn++ {
            if e.requests[f][btn] {
                return true
            }
        }
    }
    return false
}

func RequestsHere(e Elevator) bool {
    for btn := 0; btn < N_BUTTONS; btn++ {
        if e.requests[e.floor][btn] {
            return true
        }
    }
    return false
}

func RequestsChooseDirection(e Elevator) DirnBehaviourPair {
    switch e.dirn {
    case D_Up:
        if RequestsAbove(e) {
            return DirnBehaviourPair{D_Up, EB_Moving}
        } else if RequestsHere(e) {
            return DirnBehaviourPair{D_Down, EB_DoorOpen}
        } else if RequestsBelow(e) {
            return DirnBehaviourPair{D_Down, EB_Moving}
        } else {
            return DirnBehaviourPair{D_Stop, EB_Idle}
        }
    case D_Down:
        if RequestsBelow(e) {
            return DirnBehaviourPair{D_Down, EB_Moving}
        } else if RequestsHere(e) {
            return DirnBehaviourPair{D_Up, EB_DoorOpen}
        } else if RequestsAbove(e) {
            return DirnBehaviourPair{D_Up, EB_Moving}
        } else {
            return DirnBehaviourPair{D_Stop, EB_Idle}
        }
    case D_Stop:
        if RequestsHere(e) {
            return DirnBehaviourPair{D_Stop, EB_DoorOpen}
        } else if RequestsAbove(e) {
            return DirnBehaviourPair{D_Up, EB_Moving}
        } else if RequestsBelow(e) {
            return DirnBehaviourPair{D_Down, EB_Moving}
        } else {
            return DirnBehaviourPair{D_Stop, EB_Idle}
        }
    default:
        return DirnBehaviourPair{D_Stop, EB_Idle}
    }
}

func RequestsShouldStop(e Elevator) bool {
    switch e.dirn {
    case D_Down:
        return e.requests[e.floor][B_HallDown] || e.requests[e.floor][B_Cab] || !RequestsBelow(e)
    case D_Up:
        return e.requests[e.floor][B_HallUp] || e.requests[e.floor][B_Cab] || !RequestsAbove(e)
    case D_Stop:
        fallthrough
    default:
        return true
    }
}

func RequestsShouldClearImmediately(e Elevator, btnFloor int, btnType Button) bool {
    switch e.config.clearRequestVariant {
    case CV_All:
        return e.floor == btnFloor
    case CV_InDirn:
        return e.floor == btnFloor && ((e.dirn == D_Up && btnType == B_HallUp) || (e.dirn == D_Down && btnType == B_HallDown) || e.dirn == D_Stop || btnType == B_Cab)
    default:
        return false
    }
}

func RequestsClearAtCurrentFloor(e Elevator) Elevator {
    switch e.config.clearRequestVariant {
    case CV_All:
        for btn := 0; btn < N_BUTTONS; btn++ {
            e.requests[e.floor][btn] = false
        }
    case CV_InDirn:
        e.requests[e.floor][B_Cab] = false
        switch e.dirn {
        case D_Up:
            if !requestsAbove(e) && !e.requests[e.floor][B_HallUp] {
                e.requests[e.floor][B_HallDown] = false
            }
            e.requests[e.floor][B_HallUp] = false
        case D_Down:
            if !requestsBelow(e) && !e.requests[e.floor][B_HallDown] {
                e.requests[e.floor][B_HallUp] = false
            }
            e.requests[e.floor][B_HallDown] = false
        case D_Stop:
            fallthrough
        default:
            e.requests[e.floor][B_HallUp] = false
            e.requests[e.floor][B_HallDown] = false
        }
    }
    return e
}
