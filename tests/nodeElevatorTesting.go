package tests

import (
	"testing"
	"time"

	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"elev/single_elevator_algo"
)

// Communication test between a node and an elevator, from the perspective of the node. Rx and Tx as seen from the node.
func TestNodeReceivesHallButtonAndProcessesMasterAssignment(t *testing.T) {
	// Create buffered channels for communication between a node and an elevator.
	ElevatorHallButtonEventRx := make(chan elevator.ButtonEvent, 10)       // Receive hall button events from the elevator.
	ElevatorHRAStatesRx := make(chan hallRequestAssigner.HRAElevState, 10) // Receive HRAElevState updates from the elevator.
	ElevatorHallButtonEventTx := make(chan elevator.ButtonEvent, 10)       // Transmit hall button events to the elevator.
	// ElevatorHRAStatesTx := make(chan hallRequestAssigner.HRAElevState, 10) // Transmit HRAElevState updates to the elevator.

	// Run the SingleElevatorProgram in its own goroutine.
	go single_elevator_algo.SingleElevatorProgram(ElevatorHallButtonEventRx, ElevatorHRAStatesRx, ElevatorHallButtonEventTx)

	// Simulate a hall button press from the local elevator. In the program, hall calls are forwarded
	// so the elevator does not process them directly.
	println("Sending hall calls to node for processing")
	testHallEvent := elevator.ButtonEvent{
		Floor:  2,
		Button: elevator.BT_HallUp,
	}
	ElevatorHallButtonEventRx <- testHallEvent // Send the hall call to the node.
	time.Sleep(100 * time.Millisecond)

	// Wait to receive an HRAElevState update from the elevator.
	select {
	case state := <-ElevatorHRAStatesRx:
		println("Received HRAElevState")
		// Since the elevator is newly initialized, the floor should remain at -1.
		if state.Floor != -1 {
			t.Errorf("Expected initial floor -1, got %d", state.Floor)
		}
	case <-time.After(1 * time.Second):
		t.Error("Did not receive HRAElevState within timeout after hall call")
	}

	// Now simulate an incoming hall button assignment from the node.
	// Cab requests are still processed directly.
	println("Sending hall assignment to elevator")
	testCabEvent := elevator.ButtonEvent{
		Floor:  2,
		Button: elevator.BT_HallDown,
	}
	// Transmit the hall assignment to the elevator.
	ElevatorHallButtonEventTx <- testCabEvent
	time.Sleep(100 * time.Millisecond)

	// // Wait for the HRAElevState update and check that incoming hall assignment is registered.
	// select {
	// case state := <-ElevatorHRAStatesRx:
	// 	println("Received HRAElevState")
	// case <-time.After(1 * time.Second):
	// 	t.Error("Did not receive HRAElevState within timeout after hall assignment")
	// }

	time.Sleep(1 * time.Second)
}
