package elevator_fsm

import (
	"elev/elevator"
	"elev/util/config"
	"elev/util/timer"
	"fmt"
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
	FsmOnInitBetweenFloors()
}

func FsmOnInitBetweenFloors() {
	elevator.SetMotorDirection(elevator.DirectionDown)
	elev.Dir = elevator.DirectionDown
	elev.Behavior = elevator.Moving
}

func FsmOnRequestButtonPress(btnFloor int, btnType elevator.ButtonType, doorOpenTimer *timer.Timer) {
	fmt.Printf("\n\n%s(%d, %s)\n", "fsmOnRequestButtonPress", btnFloor, btnType.String())
	elevator.PrintElevator(elev)

	switch elev.Behavior {
	case elevator.DoorOpen:
		// If the elevator is at the requested floor, the door is open, and the button is pressed again, the door should remain open.
		if elevator.RequestsShouldClearImmediately(elev, btnFloor, btnType) {
			timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
		} else {
			elev.Requests[btnFloor][btnType] = true
		}
	case elevator.Moving:
		elev.Requests[btnFloor][btnType] = true
	case elevator.Idle:
		elev.Requests[btnFloor][btnType] = true
		pair := elevator.RequestsChooseDirection(elev)
		elev.Dir = pair.Dir
		elev.Behavior = pair.Behavior
		switch pair.Behavior {
		case elevator.DoorOpen:
			elevator.SetDoorOpenLamp(true)
			timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
			updatedElev, _ := elevator.RequestsClearAtCurrentFloor(elev)
			elev = updatedElev

		case elevator.Moving:
			elevator.SetMotorDirection(elev.Dir)
		case elevator.Idle:
		}
	}

	elevator.SetAllLights(&elev)

	fmt.Println("\nNew state:")
	elevator.PrintElevator(elev)
}

func FsmSetObstruction(isObstructed bool) {
	elev.IsObstructed = isObstructed
}

func FsmOnFloorArrival(newFloor int, doorOpenTimer *timer.Timer) []elevator.ButtonEvent {

	// rememmber and return the events cleared if the elevator stopped
	var clearedRequests []elevator.ButtonEvent
	fmt.Printf("\n\n%s(%d)\n", "fsmOnFloorArrival", newFloor)
	elevator.PrintElevator(elev)

	elev.Floor = newFloor
	elevator.SetFloorIndicator(elev.Floor)

	switch elev.Behavior {
	case elevator.Moving:
		if elevator.RequestsShouldStop(elev) {
			var updatedElev elevator.Elevator

			elevator.SetMotorDirection(elevator.DirectionStop)
			elevator.SetDoorOpenLamp(true)

			updatedElev, clearedRequests = elevator.RequestsClearAtCurrentFloor(elev)
			elev = updatedElev

			timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
			elevator.SetAllLights(&elev)
			elev.Behavior = elevator.DoorOpen
		}
	default:
	}

	fmt.Println("\nNew state:")
	elevator.PrintElevator(elev)
	return clearedRequests
}

func SetHallLights(lightStates [config.NUM_FLOORS][config.NUM_BUTTONS - 1]bool) {
	elev.HallLightStates = lightStates
}

func FsmOnDoorTimeout(doorOpenTimer *timer.Timer, doorStuckTimer *timer.Timer) {
	fmt.Printf("\n\n%s()\n", "fsmOnDoorTimeout")
	elevator.PrintElevator(elev)

	switch elev.Behavior {
	case elevator.DoorOpen:
		if elev.IsObstructed {
			timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
		} else {
			timer.TimerStop(doorOpenTimer)
			// stop the doorStuckTimer!
			timer.TimerStop(doorStuckTimer)
			elevator.SetDoorOpenLamp(false)

			pair := elevator.RequestsChooseDirection(elev)
			elev.Dir = pair.Dir
			elev.Behavior = pair.Behavior

			// if pair.Behavior == elevator.Moving {
			// 	elevator.SetMotorDirection(elev.Dir)
			// }

			switch elev.Behavior {
			case elevator.DoorOpen:
				timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
				timer.TimerStart(doorStuckTimer, config.DOOR_STUCK_DURATION)

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

	fmt.Println("\nNew state:")
	elevator.PrintElevator(elev)
}
