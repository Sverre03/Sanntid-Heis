package tests

// import (
// 	"elev/elevator"
// 	"elev/util/config"
// )

// func NodeElevatorCommTest() {
// 	ElevatorHallButtonAssignmentRx := make(chan [config.NUM_FLOORS][2]bool)
// 	ElevatorHRAStatesRx := make(chan elevator.ElevatorState)
// 	ElevatorHallButtonEventTx := make(chan elevator.ButtonEvent)
// 	IsDoorStuckCh := make(chan bool)
// 	DoorStateRequestCh := make(chan bool)

// 	go ElevatorProgram(ElevatorHallButtonAssignmentRx, ElevatorHRAStatesRx, ElevatorHallButtonEventTx, IsDoorStuckCh, DoorStateRequestCh)
// 	for {
// 		select {
// 		case <-ElevatorHallButtonEventRx:
// 			// Do something
// 		case <-ElevatorHRAStatesTx:
// 			// Do something
// 		case <-IsDoorStuckCh:
// 			// Do something
// 		case <-DoorStateRequestCh:
// 			// Do something
// 		case <-ElevatorHallButtonAssignmentTx:
// 			// Do something
// 		}
// 	}
// }
