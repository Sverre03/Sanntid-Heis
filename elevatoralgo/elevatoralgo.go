package elevatoralgo

import (
	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"elev/util/config"
	"elev/util/timer"
	"time"
)

// Tx and Rx is from the view of the elevator.
func ElevatorProgram(ElevatorHallButtonEventTx chan elevator.ButtonEvent,
	ElevatorHRAStatesTx chan hallRequestAssigner.HRAElevState, ElevatorHallButtonEventRx chan elevator.ButtonEvent, IsDoorStuckCh chan bool, DoorStateRequestCh chan bool) {

	var elev elevator.Elevator = elevator.NewElevator()

	buttonEvent := make(chan elevator.ButtonEvent)
	floorEvent := make(chan int)
	doorTimeoutEvent := make(chan bool)
	doorStuckEvent := make(chan bool)
	obstructionEvent := make(chan bool)

	doorOpenTimer := timer.NewTimer()  // Used to check if the door is open (if it is not closed after a certain time, 3 seconds)
	doorStuckTimer := timer.NewTimer() // Used to check if the door is stuck (if it is not closed after a certain time, 30 seconds)

	elevator.Init("localhost:15657", config.NUM_FLOORS)
	elevator.InitFSM(elev)

	prevRequestButton := make([][]bool, config.NUM_FLOORS)
	for i := range prevRequestButton {
		prevRequestButton[i] = make([]bool, config.NUM_BUTTONS)
	}

	// Start polling routines outside the loop
	go elevator.PollButtons(buttonEvent)
	go elevator.PollFloorSensor(floorEvent)
	go elevator.PollObstructionSwitch(obstructionEvent)
	go elevator.PollTimer(doorOpenTimer, doorTimeoutEvent)
	go elevator.PollTimer(doorStuckTimer, doorStuckEvent)
	go TransmitHRAElevState(elev, ElevatorHRAStatesTx)

	for {
		select {
		case button := <-buttonEvent:
			if (button.Button == elevator.BT_HallDown) || (button.Button == elevator.BT_HallUp) {
				ElevatorHallButtonEventTx <- elevator.ButtonEvent{ // Forward the hall call to the node
					Floor:  button.Floor,
					Button: button.Button,
				}
			} else {
				elevator.FsmOnRequestButtonPress(elev, button.Floor, button.Button, &doorOpenTimer)
			}

		case button := <-ElevatorHallButtonEventRx:
			elevator.FsmOnRequestButtonPress(elev, button.Floor, button.Button, &doorOpenTimer)

		case floor := <-floorEvent:
			elevator.FsmOnFloorArrival(elev, floor, &doorOpenTimer)
		case isObstructed := <-obstructionEvent:
			elevator.FsmSetObstruction(elev, isObstructed)

		case <-doorTimeoutEvent:
			if !timer.Active(doorStuckTimer) {
				timer.TimerStart(&doorStuckTimer, config.DOOR_STUCK_DURATION)
			}
			elevator.FsmOnDoorTimeout(elev, &doorOpenTimer, &doorStuckTimer)

		case <-doorStuckEvent:
			IsDoorStuckCh <- true
		}

		time.Sleep(config.INPUT_POLL_RATE)
	}
}

// Transmit the elevator state to the node
func TransmitHRAElevState(elev elevator.Elevator, ElevatorHRAStatesRx chan hallRequestAssigner.HRAElevState) {
	for range time.Tick(config.ELEV_STATE_TRANSMIT_INTERVAL) {
		ElevatorHRAStatesRx <- hallRequestAssigner.HRAElevState{
			Behavior:    elevator.ElevatorBehaviorToString[elev.Behavior],
			Floor:       elev.Floor,
			Direction:   elevator.ElevatorDirectionToString[elev.Dir],
			CabRequests: elevator.GetCabRequestsAsHRAElevState(elev),
		}
	}
}
