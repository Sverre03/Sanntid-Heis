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
	SetMotorDirection(MD_Down)
	elev.Dir = MD_Down
	elev.Behavior = EB_Moving
}

func FsmOnRequestButtonPress(elev *Elevator, btnFloor int, btnType ButtonType, doorOpenTimer *timer.Timer) {
	fmt.Printf("\n\n%s(%d, %s)\n", "fsmOnRequestButtonPress", btnFloor, ButtonToString(btnType))
	PrintElevator(*elev)

	switch elev.Behavior {
	case EB_DoorOpen:
		// If the elevator is at the requested floor, the door is open, and the button is pressed again, the door should remain open.
		if RequestsShouldClearImmediately(*elev, btnFloor, btnType) {
			timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
		} else {
			elev.Requests[btnFloor][btnType] = true
		}
	case EB_Moving:
		elev.Requests[btnFloor][btnType] = true
	case EB_Idle:
		elev.Requests[btnFloor][btnType] = true
		pair := RequestsChooseDirection(*elev)
		elev.Dir = pair.Dir
		elev.Behavior = pair.Behavior
		switch pair.Behavior {
		case EB_DoorOpen:
			SetDoorOpenLamp(true)
			timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
			*elev = RequestsClearAtCurrentFloor(*elev)
		case EB_Moving:
			SetMotorDirection(elev.Dir)
		case EB_Idle:
		}
	}

	SetAllLights(elev)

	fmt.Println("\nNew state:")
	PrintElevator(*elev)
}

func FsmSetObstruction(elev *Elevator, isObstructed bool) {
	elev.IsObstructed = isObstructed
}

func FsmOnFloorArrival(elev *Elevator, newFloor int, doorOpenTimer *timer.Timer) {
	fmt.Printf("\n\n%s(%d)\n", "fsmOnFloorArrival", newFloor)
	PrintElevator(*elev)

	elev.Floor = newFloor
	SetFloorIndicator(elev.Floor)

	switch elev.Behavior {
	case EB_Moving:
		if RequestsShouldStop(*elev) {
			SetMotorDirection(MD_Stop)
			SetDoorOpenLamp(true)
			*elev = RequestsClearAtCurrentFloor(*elev)
			timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
			SetAllLights(elev)
			elev.Behavior = EB_DoorOpen
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
	case EB_DoorOpen:
		if elev.IsObstructed {
			timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
		} else {
			timer.TimerStop(doorOpenTimer)
			timer.TimerStop(doorStuckTimer)
			SetDoorOpenLamp(false)

			pair := RequestsChooseDirection(*elev)
			elev.Dir = pair.Dir
			elev.Behavior = pair.Behavior

			// if pair.Behavior == EB_Moving {
			// 	SetMotorDirection(elev.Dir)
			// }

			switch elev.Behavior {
			case EB_DoorOpen:
				timer.TimerStart(doorOpenTimer, config.DOOR_OPEN_DURATION)
				timer.TimerStart(doorStuckTimer, config.DOOR_STUCK_DURATION)
				*elev = RequestsClearAtCurrentFloor(*elev)
				SetAllLights(elev)
			case EB_Moving, EB_Idle:
				SetDoorOpenLamp(false)
				SetMotorDirection(elev.Dir)
			}
		}
	default:
	}

	fmt.Println("\nNew state:")
	PrintElevator(*elev)
}
