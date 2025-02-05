package fsm

import (
	"config"
	"elevator"
	"elevio"
	"fmt"
	"requests"
	"timer"
)

var elev elevator.Elevator

func SetAllLights(elev elevator.Elevator) {
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for btn := 0; btn < elevio.NumButtons; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, elev.Requests[floor][btn])
		}
	}
}

func FsmOnInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.MD_Down)
	elev.Dir = elevio.MD_Down
	elev.Behavior = elevator.EB_Moving
}

func FsmOnRequestButtonPress(btnFloor int, btnType elevio.ButtonType) {
	fmt.Printf("\n\n%s(%d, %s)\n", "fsmOnRequestButtonPress", btnFloor, elevio.ButtonToString(btnType))
	elevator.PrintElevator(elev)

	switch elev.Behavior {
	case elevator.EB_DoorOpen:
		// If the elevator is at the requested floor, the door is open, and the button is pressed again, the door should remain open.
		if requests.RequestsShouldClearImmediately(elev, btnFloor, btnType) {
			timer.TimerStart(config.DoorOpenDurationS)
		} else {
			elev.Requests[btnFloor][btnType] = true
		}
	case elevator.EB_Moving:
		elev.Requests[btnFloor][btnType] = true
	case elevator.EB_Idle:
		elev.Requests[btnFloor][btnType] = true
		pair := requests.RequestsChooseDirection(elev)
		elev.Dir = pair.Dir
		elev.Behavior = pair.Behavior
		switch pair.Behavior {
		case elevator.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			timer.TimerStart(config.DoorOpenDurationS)
			elev = requests.RequestsClearAtCurrentFloor(elev)
		case elevator.EB_Moving:
			elevio.SetMotorDirection(elev.Dir)
		case elevator.EB_Idle:
		}
	}

	SetAllLights(elev)

	fmt.Println("\nNew state:")
	elevator.PrintElevator(elev)
}

func FsmSetObstruction(isObstructed bool) {
	elev.IsObstructed = isObstructed
}

func FsmOnFloorArrival(newFloor int) {
	fmt.Printf("\n\n%s(%d)\n", "fsmOnFloorArrival", newFloor)
	elevator.PrintElevator(elev)

	elev.Floor = newFloor

	elevio.SetFloorIndicator(elev.Floor)

	switch elev.Behavior {
	case elevator.EB_Moving:
		if requests.RequestsShouldStop(elev) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			elev = requests.RequestsClearAtCurrentFloor(elev)
			timer.TimerStart(config.DoorOpenDurationS)
			SetAllLights(elev)
			elev.Behavior = elevator.EB_DoorOpen
		}
	default:
	}

	fmt.Println("\nNew state:")
	elevator.PrintElevator(elev)
}

func FsmOnDoorTimeout() {
	fmt.Printf("\n\n%s()\n", "fsmOnDoorTimeout")
	elevator.PrintElevator(elev)

	switch elev.Behavior {
	case elevator.EB_DoorOpen:
		if elev.IsObstructed {
			timer.TimerStart(config.DoorOpenDurationS) // Keep the door open
		} else {
			pair := requests.RequestsChooseDirection(elev)
			elev.Dir = pair.Dir
			elev.Behavior = pair.Behavior

			switch elev.Behavior {
			case elevator.EB_DoorOpen:
				timer.TimerStart(config.DoorOpenDurationS)
				elev = requests.RequestsClearAtCurrentFloor(elev)
				SetAllLights(elev)
			case elevator.EB_Moving, elevator.EB_Idle:
				elevio.SetDoorOpenLamp(false)
				elevio.SetMotorDirection(elev.Dir)
			}
		}
	default:
	}

	fmt.Println("\nNew state:")
	elevator.PrintElevator(elev)
}
