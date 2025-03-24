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

// HandleButtonEvent is a pure function that computes state changes
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
	// fmt.Printf("\n\n%s()\n", "OnDoorTimeout")
	// elevator.PrintElevator(elev)

	switch elev.Behavior {
	case elevator.DoorOpen:
		if elev.IsObstructed {
			doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)
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

			switch elev.Behavior {
			case elevator.DoorOpen:
				doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)
				doorStuckTimer.Reset(config.DOOR_STUCK_DURATION)

				updatedElev, _ := elevator.RequestsClearAtCurrentFloor(elev)
				elev = updatedElev

				elevator.SetAllLights(&elev)
			case elevator.Moving, elevator.Idle:
				elevator.SetDoorOpenLamp(false)
				elevator.SetMotorDirection(elev.Dir)
			}
		}
	default:
	}

	// fmt.Println("\nNew state:")
	// elevator.PrintElevator(elev)
}
