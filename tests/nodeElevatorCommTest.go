package tests

import (
	"elev/elevator"
	"elev/elevatoralgo"
)

func NodeElevatorCommTest() {
	ElevatorHallButtonAssignmentTx := make(chan [config.NUM_FLOORS][2]bool)
	ElevatorHRAStatesTx := make(chan elevator.ElevatorState)
	ElevatorHallButtonEventRx := make(chan elevator.ButtonEvent)
	IsDoorStuckCh := make(chan bool)
	DoorStateRequestCh := make(chan bool)

	go elevatoralgo.ElevatorProgram(ElevatorHallButtonAssignmentTx, ElevatorHRAStatesTx, ElevatorHallButtonEventRx, IsDoorStuckCh, DoorStateRequestCh)
	for {
		select {
		case <-ElevatorHallButtonEventRx:
			// Do something
		case <-ElevatorHRAStatesTx:
			// Do something
		case <-IsDoorStuckCh:
			// Do something
		case <-DoorStateRequestCh:
			// Do something
		case <-ElevatorHallButtonAssignmentTx:
			// Do something
		}
	}
}
