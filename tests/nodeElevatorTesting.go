package tests

import (
	"testing"
	"time"

	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"elev/single_elevator_algo"
)

func TestNodeReceivesHallButtonAndProcessesMasterAssignment(t *testing.T) {
	// Create buffered channels for communication
	hallButtonEventTx := make(chan elevator.ButtonEvent, 10)
	hallButtonEventRx := make(chan elevator.ButtonEvent, 10)
	hraStatesTx := make(chan hallRequestAssigner.HRAElevState, 10)

	// Run the SingleElevatorProgram in a separate goroutine.
	go single_elevator_algo.SingleElevatorProgram(hallButtonEventTx, hraStatesTx, hallButtonEventRx)

	// Simulate a hall button press (hall call). In our program, hall calls are forwarded
	// so the elevator does not process them directly.
	testHallEvent := elevator.ButtonEvent{
		Floor:  2,
		Button: elevator.BT_HallUp,
	}
	// Send the hall event via the RX channel.
	hallButtonEventRx <- testHallEvent

	// Wait to receive an HRAElevState update.
	select {
	case state := <-hraStatesTx:
		// Since the elevator is newly initialized, the floor should remain at -1.
		if state.Floor != -1 {
			t.Errorf("Expected initial floor -1, got %d", state.Floor)
		}
	case <-time.After(2 * time.Second):
		t.Error("Did not receive HRAElevState within timeout after hall call")
	}

	// Now simulate an incoming hall button assignment from a master node.
	// In our design, non-hall events (e.g. cab requests) are processed directly.
	testCabEvent := elevator.ButtonEvent{
		Floor:  3,
		Button: elevator.BT_Cab,
	}
	// Send the cab event via the RX channel.
	hallButtonEventRx <- testCabEvent

	// Wait for the HRAElevState update and check that the cab request registers.
	select {
	case state := <-hraStatesTx:
		if !state.CabRequests[3] {
			t.Errorf("Expected cab request on floor 3 to be true")
		}
	case <-time.After(2 * time.Second):
		t.Error("Did not receive HRAElevState update for cab request within timeout")
	}

	time.Sleep(1 * time.Second)
}
