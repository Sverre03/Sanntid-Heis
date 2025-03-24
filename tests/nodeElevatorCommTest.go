package tests

// import (
// 	"elev/elevator"
// 	"elev/config"
// )

// func NodeElevatorCommTest() {
// 	ElevatorHallButtonAssignmentRx := make(chan [config.NUM_FLOORS][2]bool)
// 	ElevatorHRAStatesRx := make(chan elevator.ElevatorState)
// 	ElevatorHallButtonEventTx := make(chan elevator.ButtonEvent)
// 	DoorIsStuckCh := make(chan bool)
// 	DoorStateRequestCh := make(chan bool)

// 	go ElevatorProgram(ElevatorHallButtonAssignmentRx, ElevatorHRAStatesRx, ElevatorHallButtonEventTx, DoorIsStuckCh, DoorStateRequestCh)
// 	for {
// 		select {
// 		case <-ElevatorHallButtonEventRx:
// 			// Do something
// 		case <-ElevatorHRAStatesTx:
// 			// Do something
// 		case <-DoorIsStuckCh:
// 			// Do something
// 		case <-DoorStateRequestCh:
// 			// Do something
// 		case <-ElevatorHallButtonAssignmentTx:
// 			// Do something
// 		}
// 	}
// }
