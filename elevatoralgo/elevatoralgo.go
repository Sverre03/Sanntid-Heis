package elevatoralgo

import (
	"elev/elevator"
	"elev/util/config"
	"elev/util/timer"
	"fmt"
	"time"
)

// Tx and Rx is from the view of the elevator.
func ElevatorProgram(ElevatorHallButtonEventTx chan elevator.ButtonEvent,
	ElevatorHRAStatesTx chan elevator.ElevatorState, ElevatorHallButtonEventRx chan elevator.ButtonEvent, IsDoorStuckCh chan bool, DoorStateRequestCh chan bool) {

	var elev elevator.Elevator = elevator.NewElevator()

	buttonEvent := make(chan elevator.ButtonEvent)
	floorEvent := make(chan int)
	doorTimeoutEvent := make(chan bool)
	doorStuckEvent := make(chan bool)
	obstructionEvent := make(chan bool)

	doorOpenTimer := timer.NewTimer()  // Used to check if the door is open (if it is not closed after a certain time, 3 seconds)
	doorStuckTimer := timer.NewTimer() // Used to check if the door is stuck (if it is not closed after a certain time, 30 seconds)

	elevator.Init("localhost:15657", config.NUM_FLOORS)
	elevator.InitFSM(&elev)

	prevRequestButton := make([][]bool, config.NUM_FLOORS)
	for i := range prevRequestButton {
		prevRequestButton[i] = make([]bool, config.NUM_BUTTONS)
	}

	// Start polling routines outside the loop
	fmt.Println("Starting polling routines")
	go elevator.PollButtons(buttonEvent)
	go elevator.PollFloorSensor(floorEvent)
	go elevator.PollObstructionSwitch(obstructionEvent)
	go TransmitHRAElevState(&elev, ElevatorHRAStatesTx)

	// Check if the door has been open for its maximum duration
	go func() {
		for range time.Tick(config.INPUT_POLL_RATE) {
			if doorOpenTimer.Active && timer.TimerTimedOut(doorOpenTimer) {
				fmt.Println("BACKUP CHECK: Door timer timed out but no event received!")
				doorTimeoutEvent <- true
			}
		}
	}()

	// Check if the door is stuck
	go func() {
		for range time.Tick(50 * time.Millisecond) {
			if doorStuckTimer.Active && timer.TimerTimedOut(doorStuckTimer) {
				fmt.Println("BACKUP CHECK: Door stuck timer timed out but no event received!")
				doorStuckEvent <- true
			}
		}
	}()

	for {
		select {
		case button := <-buttonEvent:
			fmt.Printf("Button press detected: Floor %d, Button %s\n",
				button.Floor, elevator.ButtonToString(button.Button))

			if (button.Button == elevator.BT_HallDown) || (button.Button == elevator.BT_HallUp) {
				fmt.Printf("Forwarding hall call to node: Floor %d, Button %s\n",
					button.Floor, elevator.ButtonToString(button.Button))
				ElevatorHallButtonEventTx <- elevator.ButtonEvent{ // Forward the hall call to the node
					Floor:  button.Floor,
					Button: button.Button,
				}
			} else {
				elevator.FsmOnRequestButtonPress(&elev, button.Floor, button.Button, &doorOpenTimer)
			}

		case button := <-ElevatorHallButtonEventRx:
			fmt.Printf("Received hall button assignment: Floor %d, Button %s\n",
				button.Floor, elevator.ButtonToString(button.Button))
			elevator.FsmOnRequestButtonPress(&elev, button.Floor, button.Button, &doorOpenTimer)

		case floor := <-floorEvent:
			fmt.Printf("Floor sensor triggered: %d\n", floor)
			elevator.FsmOnFloorArrival(&elev, floor, &doorOpenTimer)

		case isObstructed := <-obstructionEvent:
			fmt.Printf("Obstruction state changed: %v\n", isObstructed)
			elevator.FsmSetObstruction(&elev, isObstructed)

		case <-doorTimeoutEvent:
			fmt.Println("Door timeout event detected")
			if !timer.Active(doorStuckTimer) {
				timer.TimerStart(&doorStuckTimer, config.DOOR_STUCK_DURATION)
			}
			elevator.FsmOnDoorTimeout(&elev, &doorOpenTimer, &doorStuckTimer)

		case <-doorStuckEvent:
			fmt.Println("Door stuck event detected - door has been open too long")
			IsDoorStuckCh <- true

		case <-time.After(config.INPUT_POLL_RATE):
			// To avoid blocking
		}
	}
}

// Transmit the elevator state to the node
func TransmitHRAElevState(elev *elevator.Elevator, ElevatorHRAStatesRx chan elevator.ElevatorState) {
	for range time.Tick(config.ELEV_STATE_TRANSMIT_INTERVAL) {
		ElevatorHRAStatesRx <- elevator.ElevatorState{
			Behavior:    elev.Behavior,
			Floor:       elev.Floor,
			Direction:   elev.Dir,
			CabRequests: elevator.GetCabRequestsAsElevState(*elev),
		}
	}
}
