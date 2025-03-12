package elevator

import (
	"elev/util/config"
	"elev/util/timer"
	"fmt"
)

func InitFSM(elev *Elevator) {
	for floor := 0; floor < config.NUM_FLOORS; floor++ {
		for btn := 0; btn < config.NUM_BUTTONS; btn++ {
			SetButtonLamp(ButtonType(btn), floor, false)
		}
	}
	FsmOnInitBetweenFloors(elev)
}

func SetAllLights(elev *Elevator) {
	for floor := 0; floor < config.NUM_FLOORS; floor++ {
		for btn := 0; btn < config.NUM_BUTTONS; btn++ {
			SetButtonLamp(ButtonType(btn), floor, elev.Requests[floor][btn])
		}
	}
}

func FsmOnInitBetweenFloors(elev *Elevator) {
	SetMotorDirection(DirectionDown)
	elev.Dir = DirectionDown
	elev.Behavior = Moving
}

func FsmOnRequestButtonPress(elev *Elevator, btnFloor int, btnType ButtonType, doorOpenTimer *timer.Timer) {
	fmt.Printf("\n\n%s(%d, %s)\n", "fsmOnRequestButtonPress", btnFloor, btnType.String())
	PrintElevator(*elev)

	switch elev.Behavior {
	case DoorOpen:
		// If the elevator is at the requested floor, the door is open, and the button is pressed again, the door should remain open.
		if RequestsShouldClearImmediately(*elev, btnFloor, btnType) {
			timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
		} else {
			elev.Requests[btnFloor][btnType] = true
		}
	case Moving:
		elev.Requests[btnFloor][btnType] = true
	case Idle:
		elev.Requests[btnFloor][btnType] = true
		pair := RequestsChooseDirection(*elev)
		elev.Dir = pair.Dir
		elev.Behavior = pair.Behavior
		switch pair.Behavior {
		case DoorOpen:
			SetDoorOpenLamp(true)
			timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
			updatedElev, _ := RequestsClearAtCurrentFloor(*elev)
			*elev = updatedElev

		case Moving:
			SetMotorDirection(elev.Dir)
		case Idle:
		}
	}

	SetAllLights(elev)

	fmt.Println("\nNew state:")
	PrintElevator(*elev)
}

func FsmSetObstruction(elev *Elevator, isObstructed bool) {
	elev.IsObstructed = isObstructed
}

func FsmOnFloorArrival(elev *Elevator, newFloor int, doorOpenTimer *timer.Timer, ElevatorHallAssignmentCompleteTx chan ButtonEvent) {
	fmt.Printf("\n\n%s(%d)\n", "fsmOnFloorArrival", newFloor)
	PrintElevator(*elev)

	elev.Floor = newFloor
	SetFloorIndicator(elev.Floor)

	switch elev.Behavior {
	case Moving:
		if RequestsShouldStop(*elev) {
			SetMotorDirection(DirectionStop)
			SetDoorOpenLamp(true)

			updatedElev, clearedRequests := RequestsClearAtCurrentFloor(*elev)
			*elev = updatedElev

			for _, request := range clearedRequests {
				fmt.Printf("Sending hall assignment complete: Floor %d, Button %s\n", request.Floor, request.Button.String())
				ElevatorHallAssignmentCompleteTx <- request
			}

			timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
			SetAllLights(elev)
			elev.Behavior = DoorOpen
		}
	default:
	}

	fmt.Println("\nNew state:")
	PrintElevator(*elev)
}

func FsmOnDoorTimeout(elev *Elevator, doorOpenTimer *timer.Timer, doorStuckTimer *timer.Timer) {
	fmt.Printf("\n\n%s()\n", "fsmOnDoorTimeout")
	PrintElevator(*elev)

	switch elev.Behavior {
	case DoorOpen:
		if elev.IsObstructed {
			timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
		} else {
			timer.TimerStop(doorOpenTimer)
			timer.TimerStop(doorStuckTimer)
			SetDoorOpenLamp(false)

			pair := RequestsChooseDirection(*elev)
			elev.Dir = pair.Dir
			elev.Behavior = pair.Behavior

			// if pair.Behavior == Moving {
			// 	SetMotorDirection(elev.Dir)
			// }

			switch elev.Behavior {
			case DoorOpen:
				timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
				timer.TimerStart(doorStuckTimer, config.DOOR_STUCK_DURATION)

				updatedElev, _ := RequestsClearAtCurrentFloor(*elev)
				*elev = updatedElev
				
				SetAllLights(elev)
			case Moving, Idle:
				SetDoorOpenLamp(false)
				SetMotorDirection(elev.Dir)
			}
		}
	default:
	}

	fmt.Println("\nNew state:")
	PrintElevator(*elev)
}
