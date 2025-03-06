package main

import (
	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"elev/elevatoralgo"
)

func main() {
	// tests.TestTransmitFunctions()
	// tests.RunTestNode()
	ElevatorHallButtonEventTx := make(chan elevator.ButtonEvent)
	ElevatorHRAStatesTx := make(chan elevator.ElevatorState)
	ElevatorHallButtonEventRx := make(chan elevator.ButtonEvent)
	IsDoorStuckCh := make(chan bool)
	DoorStateRequestCh := make(chan bool)
	elevatoralgo.ElevatorProgram(ElevatorHallButtonEventTx, ElevatorHRAStatesTx, ElevatorHallButtonEventRx, IsDoorStuckCh, DoorStateRequestCh)
	// Read from the channels to prevent the program from exiting and to keep the polling routines running
	for {
		select {
		case <-ElevatorHallButtonEventTx:
		case <-ElevatorHRAStatesTx:
		case <-ElevatorHallButtonEventRx:
		case <-IsDoorStuckCh:
		case <-DoorStateRequestCh:
		}
	}
}
