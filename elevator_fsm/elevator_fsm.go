package elevator_fsm

import (
	"elev/config"
	"elev/elevator"
	"fmt"
	"time"
)

var elev elevator.Elevator

func GetElevator() elevator.Elevator {
	return elev
}

func InitFSM() {
	elev = elevator.NewElevator()

	for floor := range config.NUM_FLOORS {
		for btn := range config.NUM_BUTTONS {
			elevator.SetButtonLamp(elevator.ButtonType(btn), floor, false)
		}
	}
	OnInitBetweenFloors()
}

func OnInitBetweenFloors() {
	elevator.SetMotorDirection(elevator.DirectionDown)
	elev.Dir = elevator.DirectionDown
	elev.Behavior = elevator.Moving
}

func ClearHallAssignments(newHallAssignments [config.NUM_FLOORS][2]bool) bool {
	shouldStop := false
	for floor := range config.NUM_FLOORS {
		for btn := range 2 {
			if elev.Requests[floor][btn] && !newHallAssignments[floor][btn] {
				fmt.Printf("Hall assignment removed at floor %d, button %d\n", floor, btn)
				elev.Requests[floor][btn] = false
				shouldStop = true
			}
		}
	}
	return shouldStop
}

func StopElevator() {
	elevator.SetMotorDirection(elevator.DirectionStop)
	elev.Dir = elevator.DirectionStop
	elev.Behavior = elevator.Idle
}

func OnRequestButtonPress(btnFloor int, btnType elevator.ButtonType, doorOpenTimer *time.Timer) {
	// Compute new elevator state
	newState, resetDoorTimer := HandleButtonEvent(btnFloor, btnType, doorOpenTimer)

	// Apply side effects
	if resetDoorTimer {
		doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)
		elevator.SetDoorOpenLamp(true)
	}

	if newState.Behavior == elevator.Moving && elev.Behavior != elevator.Moving {
		elevator.SetMotorDirection(newState.Dir)
	}

	elev = newState
	elevator.SetAllLights(&elev)
}

// HandleButtonEvent is a pure function that computes state changes
func HandleButtonEvent(btnFloor int, btnType elevator.ButtonType, doorOpenTimer *time.Timer) (elevator.Elevator, bool) {

	newState := elev
	resetDoorTimer := false

	// If the elevator is idle and the button is pressed in the same floor, the door should remain open
	switch newState.Behavior {
	case elevator.DoorOpen:
		// If the elevator is at the requested floor, the door is open, and the button is pressed again, the door should remain open.
		if elevator.RequestsShouldClearImmediately(newState, btnFloor, btnType) {
			resetDoorTimer = true
		} else {
			newState.Requests[btnFloor][btnType] = true
		}
	case elevator.Moving:
		newState.Requests[btnFloor][btnType] = true
	case elevator.Idle:
		newState.Requests[btnFloor][btnType] = true
		pair := elevator.RequestsChooseDirection(newState)
		newState.Dir = pair.Dir
		newState.Behavior = pair.Behavior

		switch pair.Behavior {
		case elevator.DoorOpen:
			resetDoorTimer = true
			newState = elevator.RequestsClearAtCurrentFloor(newState)
		case elevator.Moving, elevator.Idle:
			// do nothing
		}
	}

	return newState, resetDoorTimer
}

func SetObstruction(isObstructed bool) {
	elev.IsObstructed = isObstructed
}

func OnFloorArrival(newFloor int, doorOpenTimer *time.Timer) {
	elev.Floor = newFloor
	elevator.SetFloorIndicator(elev.Floor)

	switch elev.Behavior {
	case elevator.Moving:
		if elevator.RequestsShouldStop(elev) {
			elevator.SetMotorDirection(elevator.DirectionStop)
			elevator.SetDoorOpenLamp(true)
			doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)

			elev = elevator.RequestsClearAtCurrentFloor(elev)
			elevator.SetButtonLamp(elevator.ButtonCab, elev.Floor, false)

			elev.Behavior = elevator.DoorOpen
		}
	default:
	}
}

func SetHallLights(lightStates [config.NUM_FLOORS][config.NUM_BUTTONS - 1]bool) {
	elev.HallLightStates = lightStates
	elevator.SetAllLights(&elev)
}

func OnDoorTimeout(doorOpenTimer *time.Timer, doorStuckTimer *time.Timer) {
	switch elev.Behavior {
	case elevator.DoorOpen:
		if elev.IsObstructed {
			doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)
			if !elev.DoorStuckTimerActive {
				doorStuckTimer.Reset(config.DOOR_STUCK_DURATION)
				elev.DoorStuckTimerActive = true
				elevator.SetStopLamp(true)
			}
		} else {
			doorStuckTimer.Stop()
			elevator.SetDoorOpenLamp(false)

			pair := elevator.RequestsChooseDirection(elev)
			elev.Dir = pair.Dir
			elev.Behavior = pair.Behavior

			elev = elevator.RequestsClearAtCurrentFloor(elev)

			switch elev.Behavior {
			case elevator.DoorOpen:
				doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)
				doorStuckTimer.Reset(config.DOOR_STUCK_DURATION)

				elev = elevator.RequestsClearAtCurrentFloor(elev)

				elevator.SetAllLights(&elev)

			case elevator.Moving, elevator.Idle:
				elevator.SetDoorOpenLamp(false)
				elevator.SetMotorDirection(elev.Dir)
			}
		}
	default:
	}
}
