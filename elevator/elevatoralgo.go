package elevator

import (
	"elev/util/config"
	"elev/util/timer"
	"fmt"
	"time"
)

// ElevatorProgram operates a single elevator
// It manages the elevator state machine, events from hardware,
// and communicates with the hall request assigner.
func ElevatorProgram(
	ElevatorHallButtonEventTx chan ButtonEvent,
	ElevatorHRAStatesTx chan ElevatorState,
	ElevatorHallButtonAssignmentRx chan [config.NUM_FLOORS][2]bool,
	IsDoorStuckCh chan bool,
	DoorStateRequestCh chan bool) {

	// Initialize the elevator
	elev := NewElevator()
	Init("localhost:15657", config.NUM_FLOORS)
	InitFSM(&elev)

	// Channels for events
	buttonEvent := make(chan ButtonEvent)
	floorEvent := make(chan int)
	doorTimeoutEvent := make(chan bool)
	doorStuckEvent := make(chan bool)
	obstructionEvent := make(chan bool)

	doorOpenTimer := timer.NewTimer()  // Used to check if the door is open (if it is not closed after a certain time, 3 seconds)
	doorStuckTimer := timer.NewTimer() // Used to check if the door is stuck (if it is not closed after a certain time, 30 seconds)

	fmt.Println("Starting polling routines")
	go PollButtons(buttonEvent)
	go PollFloorSensor(floorEvent)
	go PollObstructionSwitch(obstructionEvent)

	go TransmitHRAElevState(&elev, ElevatorHRAStatesTx) // Transmits the elevator state to the node periodically

	go func() {
		for range time.Tick(config.INPUT_POLL_RATE) {
			if doorOpenTimer.Active && timer.TimerTimedOut(doorOpenTimer) { // Check if the door has been open for its maximum duration (3 seconds)
				fmt.Println("Door timer timed out")
				doorTimeoutEvent <- true
			}
		}
	}()

	go func() {
		for range time.Tick(config.INPUT_POLL_RATE) {
			if doorStuckTimer.Active && timer.TimerTimedOut(doorStuckTimer) { // Check if the door is stuck (being open for more than 30 seconds)
				fmt.Println("Door stuck timer timed out!")
				doorStuckEvent <- true
			}
		}
	}()

	for {
		select {
		case button := <-buttonEvent:
			fmt.Printf("Button press detected: Floor %d, Button %s\n",
				button.Floor, ButtonToString(button.Button))

			if (button.Button == BT_HallDown) || (button.Button == BT_HallUp) {
				fmt.Printf("Forwarding hall call to node: Floor %d, Button %s\n",
					button.Floor, ButtonToString(button.Button))
				ElevatorHallButtonEventTx <- ButtonEvent{ // Forward the hall call to the node
					Floor:  button.Floor,
					Button: button.Button,
				}
			} else {
				FsmOnRequestButtonPress(&elev, button.Floor, button.Button, &doorOpenTimer)
			}

		case hallButtons := <-ElevatorHallButtonAssignmentRx:
			fmt.Printf("Received hall button assignment")
			for floor := 0; floor < config.NUM_FLOORS; floor++ {
				for hallButton := 0; hallButton < 2; hallButton++ {
					elev.Requests[floor][hallButton] = hallButtons[floor][hallButton]
					FsmOnRequestButtonPress(&elev, floor, ButtonType(hallButton), &doorOpenTimer)
				}
			}
			SetAllLights(&elev)

		case floor := <-floorEvent:
			fmt.Printf("Floor sensor triggered: %d\n", floor)
			FsmOnFloorArrival(&elev, floor, &doorOpenTimer)

		case isObstructed := <-obstructionEvent:
			fmt.Printf("Obstruction state changed: %v\n", isObstructed)
			FsmSetObstruction(&elev, isObstructed)

		case <-doorTimeoutEvent:
			fmt.Println("Door timeout event detected")
			if !timer.Active(doorStuckTimer) {
				timer.TimerStart(&doorStuckTimer, config.DOOR_STUCK_DURATION)
			}
			FsmOnDoorTimeout(&elev, &doorOpenTimer, &doorStuckTimer)

		case <-doorStuckEvent:
			fmt.Println("Door stuck event detected - door has been open too long")
			IsDoorStuckCh <- true

		case <-time.After(config.INPUT_POLL_RATE):
			// To avoid blocking
		}
	}
}

// Transmit the elevator state to the node
func TransmitHRAElevState(elev *Elevator, ElevatorHRAStatesRx chan ElevatorState) {
	for range time.Tick(config.ELEV_STATE_TRANSMIT_INTERVAL) {
		ElevatorHRAStatesRx <- ElevatorState{
			Behavior:    elev.Behavior,
			Floor:       elev.Floor,
			Direction:   elev.Dir,
			CabRequests: GetCabRequestsAsElevState(*elev),
		}
	}
}
