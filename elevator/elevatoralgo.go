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
	ElevatorStateTx chan ElevatorState,
	ElevatorHallAssignmentRx chan [config.NUM_FLOORS][2]bool,
	ElevatorHallAssignmentCompleteTx chan ButtonEvent,
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

	startHardwarePolling(buttonEvent, floorEvent, obstructionEvent)

	go transmitElevatorState(&elev, ElevatorStateTx) // Transmits the elevator state to the node periodically

	startTimerMonitoring(&doorOpenTimer, &doorStuckTimer, doorTimeoutEvent, doorStuckEvent)

	for {
		select {
		case button := <-buttonEvent:
			handleButtonEvent(&elev, button, ElevatorHallButtonEventTx, &doorOpenTimer)

		case hallButtons := <-ElevatorHallAssignmentRx:
			AssignHallButtons(&elev, hallButtons, &doorOpenTimer)

		case floor := <-floorEvent:
			FsmOnFloorArrival(&elev, floor, &doorOpenTimer)

		case isObstructed := <-obstructionEvent:
			FsmSetObstruction(&elev, isObstructed)

		case <-doorTimeoutEvent:
			handleDoorTimeout(&elev, &doorOpenTimer, &doorStuckTimer)

		case <-doorStuckEvent:
			IsDoorStuckCh <- true

		case <-time.After(config.INPUT_POLL_RATE):
		}
	}
}

func startHardwarePolling(buttonEvent chan ButtonEvent, floorEvent chan int, obstructionEvent chan bool) {
	fmt.Println("Starting polling routines")
	go PollButtons(buttonEvent)
	go PollFloorSensor(floorEvent)
	go PollObstructionSwitch(obstructionEvent)
}

// startTimerMonitoring sets up goroutines to monitor timer events
func startTimerMonitoring(doorOpenTimer *timer.Timer, doorStuckTimer *timer.Timer, doorTimeoutEvent chan bool, doorStuckEvent chan bool) {
	// Monitor door open timeout (3 seconds)
	go func() {
		for range time.Tick(config.INPUT_POLL_RATE) {
			if doorOpenTimer.Active && timer.TimerTimedOut(*doorOpenTimer) {
				fmt.Println("Door timer timed out")
				doorTimeoutEvent <- true
			}
		}
	}()

	// Monitor door stuck timeout (30 seconds)
	go func() {
		for range time.Tick(config.INPUT_POLL_RATE) {
			if doorStuckTimer.Active && timer.TimerTimedOut(*doorStuckTimer) {
				fmt.Println("Door stuck timer timed out!")
				doorStuckEvent <- true
			}
		}
	}()
}

// Transmit the elevator state to the node
func transmitElevatorState(elev *Elevator, ElevatorStateRx chan ElevatorState) {
	for range time.Tick(config.ELEV_STATE_TRANSMIT_INTERVAL) {
		ElevatorStateRx <- ElevatorState{
			Behavior:    elev.Behavior,
			Floor:       elev.Floor,
			Direction:   elev.Dir,
			CabRequests: GetCabRequestsAsElevState(*elev),
		}
	}
}

func handleButtonEvent(elev *Elevator, button ButtonEvent, ElevatorHallButtonEventTx chan ButtonEvent, doorOpenTimer *timer.Timer) {
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
		FsmOnRequestButtonPress(elev, button.Floor, button.Button, doorOpenTimer)
	}
}

func AssignHallButtons(elev *Elevator, hallButtons [config.NUM_FLOORS][2]bool, doorOpenTimer *timer.Timer) {
	fmt.Printf("Received hall button assignment")
	for floor := 0; floor < config.NUM_FLOORS; floor++ {
		for hallButton := 0; hallButton < 2; hallButton++ {
			elev.Requests[floor][hallButton] = hallButtons[floor][hallButton]
			if elev.Requests[floor][hallButton] {
				FsmOnRequestButtonPress(elev, floor, ButtonType(hallButton), doorOpenTimer)
			}
		}
	}
	SetAllLights(elev)
}

func handleDoorTimeout(elev *Elevator, doorOpenTimer *timer.Timer, doorStuckTimer *timer.Timer) {
	fmt.Println("Door timeout event detected")
	if !timer.Active(*doorStuckTimer) {
		timer.TimerStart(doorStuckTimer, config.DOOR_STUCK_DURATION)
	}
	FsmOnDoorTimeout(elev, doorOpenTimer, doorStuckTimer)
}
