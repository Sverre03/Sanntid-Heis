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

func OnRequestButtonPress(btnFloor int, btnType elevator.ButtonType, doorOpenTimer *time.Timer) {
	fmt.Printf("new local elevator assignment: %d, %s)\n", btnFloor, btnType.String())

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

func RemoveRequest(floor int, btnType elevator.ButtonType) {
	elev.Requests[floor][btnType] = false
	// elevator.SetButtonLamp(btnType, floor, false)
}

func SetObstruction(isObstructed bool) {
	elev.IsObstructed = isObstructed
}

func OnFloorArrival(newFloor int, doorOpenTimer *time.Timer) {

	// rememmber and return the events cleared if the elevator stopped
	elev.Floor = newFloor
	elevator.SetFloorIndicator(elev.Floor)

	switch elev.Behavior {
	case elevator.Moving:
		if elevator.RequestsShouldStop(elev) {
			fmt.Printf("Elevator stopped at floor %d\n", elev.Floor)
			fmt.Printf("Motor direction: %s\n", elev.Dir.String())
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
				fmt.Printf("Obstruction detected, starting door stuck timer\n")
				doorStuckTimer.Reset(config.DOOR_STUCK_DURATION)
				elev.DoorStuckTimerActive = true
				elevator.SetStopLamp(true)
			}
		} else {
			// stop the doorStuckTimer!
			doorStuckTimer.Stop()
			elevator.SetDoorOpenLamp(false)

			pair := elevator.RequestsChooseDirection(elev)
			elev.Dir = pair.Dir
			elev.Behavior = pair.Behavior

			// if pair.Behavior == elevator.Moving {
			// 	elevator.SetMotorDirection(elev.Dir)
			// }
			fmt.Printf("Motor direction: %s\n", elev.Dir.String())
			updatedElev := elevator.RequestsClearAtCurrentFloor(elev)
			elev = updatedElev

			switch elev.Behavior {
			case elevator.DoorOpen:
				doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)
				doorStuckTimer.Reset(config.DOOR_STUCK_DURATION)

				updatedElev := elevator.RequestsClearAtCurrentFloor(elev)
				elev = updatedElev

				elevator.SetAllLights(&elev)
			case elevator.Moving, elevator.Idle:
				elevator.SetDoorOpenLamp(false)
				elevator.SetMotorDirection(elev.Dir)
			}
		}
	default:
	}
}
