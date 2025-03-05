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
	obstructionEvent := make(chan bool)

	doorTimeoutTimer := timer.NewTimer()
	doorStuckTimer := timer.NewTimer()

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
	go elevator.PollTimer(doorTimeoutTimer, doorTimeoutEvent)
	go elevator.PollTimer(doorStuckTimer, DoorStateRequestCh)
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
				elevator.FsmOnRequestButtonPress(elev, button.Floor, button.Button, inTimer)
			}

		case button := <-ElevatorHallButtonEventRx:
			elevator.FsmOnRequestButtonPress(elev, button.Floor, button.Button, inTimer) // Process the hall call from the node

		case <-DoorStateRequestCh:
			IsDoorStuckCh <- elevator.IsDoorStuck(elev)

		case floor := <-floorEvent:
			elevator.FsmOnFloorArrival(elev, floor, inTimer)

		case isObstructed := <-obstructionEvent:
			elevator.FsmSetObstruction(elev, isObstructed)

		case <-doorTimeoutEvent:
			elevator.FsmOnDoorTimeout(elev, inTimer)
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
