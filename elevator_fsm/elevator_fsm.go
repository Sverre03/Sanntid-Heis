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

	for floor := 0; floor < config.NUM_FLOORS; floor++ {
		for btn := 0; btn < config.NUM_BUTTONS; btn++ {
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

func OnRequestButtonPress(btnFloor int, btnType elevator.ButtonType, doorOpenTimer *time.Timer) []elevator.ButtonEvent {
	fmt.Printf("new local elevator assignment: %d, %s)\n", btnFloor, btnType.String())

	// Compute new elevator state
	newState, clearedEvents, resetDoorTimer := HandleButtonEvent(btnFloor, btnType, doorOpenTimer)

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

	return clearedEvents
}

// HandleButtonEvent is a pure function that computes state changes.
// It is flawed though, since it has access to the global state elev, making it impure.
// It should be refactored to take the elevator state as an argument.
func HandleButtonEvent(btnFloor int, btnType elevator.ButtonType, doorOpenTimer *time.Timer) (elevator.Elevator, []elevator.ButtonEvent, bool) {

	newState := elev
	resetDoorTimer := false
	var clearedEvents []elevator.ButtonEvent

	// If the elevator is idle and the button is pressed in the same floor, the door should remain open
	switch newState.Behavior {
	case elevator.DoorOpen:
		// If the elevator is at the requested floor, the door is open, and the button is pressed again, the door should remain open.
		if elevator.RequestsShouldClearImmediately(newState, btnFloor, btnType) {
			resetDoorTimer = true
			if btnType != elevator.ButtonCab {
				clearedEvents = append(clearedEvents, elevator.ButtonEvent{Floor: btnFloor, Button: btnType})
			}
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
			updatedElev, events := elevator.RequestsClearAtCurrentFloor(newState)
			newState = updatedElev
			clearedEvents = events
		case elevator.Moving, elevator.Idle:
			// do nothing
		}
	}

	return newState, clearedEvents, resetDoorTimer
}

func RemoveRequest(floor int, btnType elevator.ButtonType) {
	elev.Requests[floor][btnType] = false
	elevator.SetButtonLamp(btnType, floor, false)
}

func SetObstruction(isObstructed bool) {
	elev.IsObstructed = isObstructed
}

func OnFloorArrival(newFloor int, doorOpenTimer *time.Timer) []elevator.ButtonEvent {

	// rememmber and return the events cleared if the elevator stopped
	var clearedRequests []elevator.ButtonEvent

	elev.Floor = newFloor
	elevator.SetFloorIndicator(elev.Floor)

	switch elev.Behavior {
	case elevator.Moving:
		if elevator.RequestsShouldStop(elev) {
			var updatedElev elevator.Elevator

			elevator.SetMotorDirection(elevator.DirectionStop)
			elevator.SetDoorOpenLamp(true)
			doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)

			updatedElev, clearedRequests = elevator.RequestsClearAtCurrentFloor(elev)
			elev = updatedElev

			elevator.SetAllLights(&elev)
			elev.Behavior = elevator.DoorOpen
		}
	default:
	}

	return clearedRequests
}

func SetHallLights(lightStates [config.NUM_FLOORS][config.NUM_BUTTONS - 1]bool) {
	elev.HallLightStates = lightStates
	elevator.SetAllLights(&elev)
}

func OnDoorTimeout(doorOpenTimer *time.Timer, doorStuckTimer *time.Timer) {
	// Calculate new state and actions (functional core)
	newState, resetDoorOpenTimer, stopDoorStuckTimer, resetDoorStuckTimer := HandleDoorTimeout(elev)

	// Apply side effects (imperative shell)
	if resetDoorOpenTimer {
		doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)
	}

	if stopDoorStuckTimer {
		doorStuckTimer.Stop()
		elevator.SetDoorOpenLamp(false)
	}

	if resetDoorStuckTimer {
		doorStuckTimer.Reset(config.DOOR_STUCK_DURATION)
	}

	// Handle motor direction changes if state changed
	if newState.Behavior != elev.Behavior {
		if newState.Behavior == elevator.Moving {
			elevator.SetMotorDirection(newState.Dir)
		} else if elev.Behavior == elevator.DoorOpen && newState.Behavior != elevator.DoorOpen {
			elevator.SetDoorOpenLamp(false)
		}
	}

	// Update the elevator state
	elev = newState

	// Update lights based on the new state
	elevator.SetAllLights(&elev)
}

// HandleDoorTimeout is also flawed, since it has access to the global state elev, making it impure.
// It should be refactored to take the elevator state as an argument.
func HandleDoorTimeout(elev elevator.Elevator) (elevator.Elevator, bool, bool, bool) {

	newState := elev
	resetDoorOpenTimer := false
	stopDoorStuckTimer := false
	resetDoorStuckTimer := false

	switch newState.Behavior {
	case elevator.DoorOpen:
		if newState.IsObstructed {
			// Door is obstructed, keep it open
			resetDoorOpenTimer = true
		} else {
			// Door can close, determine next action
			stopDoorStuckTimer = true

			// Determine next direction and behavior
			pair := elevator.RequestsChooseDirection(newState)
			newState.Dir = pair.Dir
			newState.Behavior = pair.Behavior

			switch newState.Behavior {
			case elevator.DoorOpen:
				// Door should stay open (new request at same floor)
				resetDoorOpenTimer = true
				resetDoorStuckTimer = true

				// Clear requests at the current floor
				updatedElev, _ := elevator.RequestsClearAtCurrentFloor(newState)
				newState = updatedElev
			case elevator.Moving, elevator.Idle:
				// Door should close, elevator moves or stays idle
			}
		}
	default:
		// No state change needed
	}

	return newState, resetDoorOpenTimer, stopDoorStuckTimer, resetDoorStuckTimer
}

// func OnDoorTimeout(doorOpenTimer *time.Timer, doorStuckTimer *time.Timer) {
// 	switch elev.Behavior {
// 	case elevator.DoorOpen:
// 		if elev.IsObstructed {
// 			doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)
// 		} else {
// 			// stop the doorStuckTimer!
// 			doorStuckTimer.Stop()
// 			elevator.SetDoorOpenLamp(false)

// 			pair := elevator.RequestsChooseDirection(elev)
// 			elev.Dir = pair.Dir
// 			elev.Behavior = pair.Behavior

// 			// if pair.Behavior == elevator.Moving {
// 			// 	elevator.SetMotorDirection(elev.Dir)
// 			// }

// 			switch elev.Behavior {
// 			case elevator.DoorOpen:
// 				doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)
// 				doorStuckTimer.Reset(config.DOOR_STUCK_DURATION)

// 				updatedElev, _ := elevator.RequestsClearAtCurrentFloor(elev)
// 				elev = updatedElev

// 				elevator.SetAllLights(&elev)
// 			case elevator.Moving, elevator.Idle:
// 				elevator.SetDoorOpenLamp(false)
// 				elevator.SetMotorDirection(elev.Dir)
// 			}
// 		}
// 	default:
// 	}
// }
