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
	OnInitBetweenFloors(elevator.DirectionDown)
	elevator.SetStopLamp(false)
}

func OnInitBetweenFloors(direction elevator.MotorDirection) {
	elevator.SetMotorDirection(direction)
	elev.Dir = direction
	elev.Behavior = elevator.Moving
}

func RecoverFromStuckBetweenFloors() {
	if 0 < elev.Floor && elev.Floor <= config.NUM_FLOORS-1 {
		OnInitBetweenFloors(elevator.DirectionDown)
	} else if elev.Floor == 0 {
		OnInitBetweenFloors(elevator.DirectionUp)
	}
}

// remove active hall assignments from the elevator that are not in the new hall assignments
func RemoveInvalidHallAssignments(newHallAssignments [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool) bool {
	shouldStop := false
	for floor := range config.NUM_FLOORS {
		for btn := range config.NUM_HALL_BUTTONS {
			if elev.Requests[floor][btn] && !newHallAssignments[floor][btn] {
				elev.Requests[floor][btn] = false
				shouldStop = true
			}
		}
	}
	return shouldStop
}

func StopElevator() {
	elevator.SetMotorDirection(elevator.DirectionStop)
	elev.Behavior = elevator.StoppedBetweenFloors
	elev.Dir = elevator.DirectionStop
}

func ResumeElevator() {
	pair := elevator.RequestsChooseDirection(elev)

	// Prevent out of bounds movement
	if (elev.Floor == 0 && pair.Dir == elevator.DirectionDown) ||
		(elev.Floor == config.NUM_FLOORS-1 && pair.Dir == elevator.DirectionUp) {
		fmt.Printf("Safety: Prevented invalid direction %d at floor %d\n", pair.Dir, elev.Floor)
		pair.Dir = elevator.DirectionStop
		pair.Behavior = elevator.Idle
	}

	elev.Dir = pair.Dir
	elev.Behavior = pair.Behavior
	elevator.SetMotorDirection(elev.Dir)
}

func OnRequestButtonPress(btnFloor int, btnType elevator.ButtonType, doorOpenTimer *time.Timer) {
	// Compute new elevator state
	updatedElev, resetDoorTimer := HandleButtonEvent(btnFloor, btnType, doorOpenTimer)

	// Apply side effects
	if resetDoorTimer {
		doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)
		elevator.SetDoorOpenLamp(true)
	}

	if updatedElev.Behavior == elevator.Moving && elev.Behavior != elevator.Moving {
		elevator.SetMotorDirection(updatedElev.Dir)
	}

	elev = updatedElev
	elevator.SetAllLights(&elev)
}

// Applies changes to the elevator state based on a button press event
func HandleButtonEvent(btnFloor int, btnType elevator.ButtonType, doorOpenTimer *time.Timer) (elevator.Elevator, bool) {

	updatedElev := GetElevator()
	resetDoorTimer := false

	switch updatedElev.Behavior {
	case elevator.DoorOpen:
		if elevator.RequestsShouldClearImmediately(updatedElev, btnFloor, btnType) {
			fmt.Printf("Clear request immediately %d\n", int(btnType))
			resetDoorTimer = true
		} else {
			updatedElev.Requests[btnFloor][btnType] = true
		}
	case elevator.Moving, elevator.StoppedBetweenFloors:
		updatedElev.Requests[btnFloor][btnType] = true
	case elevator.Idle:
		updatedElev.Requests[btnFloor][btnType] = true
		pair := elevator.RequestsChooseDirection(updatedElev)
		updatedElev.Dir = pair.Dir
		updatedElev.Behavior = pair.Behavior

		switch pair.Behavior {
		case elevator.DoorOpen:
			resetDoorTimer = true
			updatedElev = elevator.RequestsClearAtCurrentFloor(updatedElev)
		case elevator.Moving, elevator.Idle:
			// Do nothing
		}
	}

	return updatedElev, resetDoorTimer
}

func SetObstruction(isObstructed bool) {
	elev.IsObstructed = isObstructed
}

func OnFloorArrival(newFloor int, doorOpenTimer *time.Timer) {
	elev.Floor = newFloor
	elevator.SetFloorIndicator(elev.Floor)

	if elev.Behavior == elevator.Moving {
		if elevator.RequestsShouldStop(elev) {
			elevator.SetMotorDirection(elevator.DirectionStop)
			elevator.SetDoorOpenLamp(true)
			doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)

			elev = elevator.RequestsClearAtCurrentFloor(elev)
			elevator.SetButtonLamp(elevator.ButtonCab, elev.Floor, false)

			elev.Behavior = elevator.DoorOpen
		}
	}
}

func UpdateHallLightStates(lightStates [config.NUM_FLOORS][config.NUM_BUTTONS - 1]bool) {
	elev.HallLightStates = lightStates
	elevator.SetHallLights(elev.HallLightStates)
}

func UpdateElevStuckTimerActiveState(isActive bool) {
	elev.DoorStuckTimerActive = isActive
}

func OnDoorTimeout(doorOpenTimer *time.Timer, doorStuckTimer *time.Timer) {
	if elev.Behavior == elevator.DoorOpen {
		if elev.IsObstructed {
			doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)
			if !elev.DoorStuckTimerActive {
				doorStuckTimer.Reset(config.DOOR_STUCK_DURATION)
				elev.DoorStuckTimerActive = true
				elevator.SetStopLamp(true)
			}
		} else {
			doorStuckTimer.Stop()
			elev.DoorStuckTimerActive = false
			elevator.SetDoorOpenLamp(false)
			elevator.SetStopLamp(false)

			pair := elevator.RequestsChooseDirection(elev)
			elev.Dir = pair.Dir
			elev.Behavior = pair.Behavior

			elev = elevator.RequestsClearAtCurrentFloor(elev)

			switch elev.Behavior {
			case elevator.DoorOpen:
				doorOpenTimer.Reset(config.DOOR_OPEN_DURATION)
				doorStuckTimer.Reset(config.DOOR_STUCK_DURATION)
				elevator.SetDoorOpenLamp(true)

				elev = elevator.RequestsClearAtCurrentFloor(elev)

				elevator.SetAllLights(&elev)

			case elevator.Moving, elevator.Idle:
				elevator.SetDoorOpenLamp(false)
				elevator.SetMotorDirection(elev.Dir)
			}
		}
	}
}
