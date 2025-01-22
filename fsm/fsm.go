package fsm

import (
	"Driver-go/elevio"
	"elevator"
	"fmt"
	"requests"
	"time"
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

func fsmOnInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.MD_Down)
	elev.Dir = elevio.MD_Down
	elev.Behavior = elevator.EB_Moving
}

func fsmOnRequestButtonPress(btnFloor int, btnType elevio.ButtonType) {
	fmt.Printf("\n\n%s(%d, %s)\n", "fsmOnRequestButtonPress", btnFloor, elevio.ButtonToString(btnType))
	elevator.PrintElevator(elev)

	switch elev.Behavior {
	case elevator.EB_DoorOpen:
		if requests.ShouldClearImmediately(elev, btnFloor, btnType) {
			timer.Start(1000 * time.Millisecond) // timer.Start(elev.Config.DoorOpenDurationS)
		} else {
			elev.Requests[btnFloor][btnType] = true
		}
	case elevator.EB_Moving:
		elev.Requests[btnFloor][btnType] = true
	case elevator.EB_Idle:
		elev.Requests[btnFloor][btnType] = true
		pair := requests.ChooseDirection(elev)
		elev.Dir = pair.Dirn
		elev.Behavior = pair.Behavior
		switch pair.Behavior {
		case elevator.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			timer.Start(1000 * time.Millisecond) // timer.Start(elev.Config.DoorOpenDurationS)
			elev = requests.ClearAtCurrentFloor(elev)
		case elevator.EB_Moving:
			elevio.SetMotorDirection(elev.Dir)
		case elevator.EB_Idle:
		}
	}

	SetAllLights(elev)

	fmt.Println("\nNew state:")
	elevator.PrintElevator(elev)
}

func fsmOnFloorArrival(newFloor int) {
	fmt.Printf("\n\n%s(%d)\n", "fsmOnFloorArrival", newFloor)
	elevator.PrintElevator(elev)

	elev.Floor = newFloor

	elevio.SetFloorIndicator(elev.Floor)

	switch elev.Behavior {
	case elevator.EB_Moving:
		if requests.ShouldStop(elev) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			elev = requests.ClearAtCurrentFloor(elev)
			timer.Start(1000 * time.Millisecond) // timer.Start(elev.Config.DoorOpenDurationS)
			SetAllLights(elev)
			elev.Behavior = elevator.EB_DoorOpen
		}
	default:
	}

	fmt.Println("\nNew state:")
	elevator.PrintElevator(elev)
}

func fsmOnDoorTimeout() {
	fmt.Printf("\n\n%s()\n", "fsmOnDoorTimeout")
	elevator.PrintElevator(elev)

	switch elev.Behavior {
	case elevator.EB_DoorOpen:
		pair := requests.ChooseDirection(elev)
		elev.Dir = pair.Dirn
		elev.Behavior = pair.Behavior

		switch elev.Behavior {
		case elevator.EB_DoorOpen:
			timer.Start(1000 * time.Millisecond) // timer.Start(elev.Config.DoorOpenDurationS)
			elev = requests.ClearAtCurrentFloor(elev)
			SetAllLights(elev)
		case elevator.EB_Moving, elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elev.Dir)
		}
	default:
	}

	fmt.Println("\nNew state:")
	elevator.PrintElevator(elev)
}
