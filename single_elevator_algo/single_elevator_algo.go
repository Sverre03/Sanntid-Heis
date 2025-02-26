package single_elevator_algo

import (
	"elev/elevator"
	"elev/node"
	"elev/util/config"
	"time"
)

func SingleElevatorProgram() {
	buttonEvent := make(chan elevator.ButtonEvent)
	floorEvent := make(chan int)
	doorTimeoutEvent := make(chan bool)
	obstructionEvent := make(chan bool)

	elevator.Init("localhost:15657", config.NUM_FLOORS)
	elevator.InitFSM()

	prevRequestButton := make([][]bool, config.NUM_FLOORS)
	for i := range prevRequestButton {
		prevRequestButton[i] = make([]bool, config.NUM_BUTTONS)
	}

	// Start polling routines outside the loop
	go elevator.PollButtons(buttonEvent)
	go elevator.PollFloorSensor(floorEvent)
	go elevator.PollObstructionSwitch(obstructionEvent)
	go elevator.PollTimer(doorTimeoutEvent)
	go TransmitHRAElevState()

	for {
		select {
		case button := <-buttonEvent:
			if (button.Button == elevator.BT_HallDown) || (button.Button == elevator.BT_HallUp) {
				node.ElevatorHallButtonEventRx <- elevator.ButtonEvent{
					Floor:  button.Floor,
					Button: button.Button,
				}
			} else {
				elevator.FsmOnRequestButtonPress(button.Floor, button.Button)
			}

		case floor := <-floorEvent:
			elevator.FsmOnFloorArrival(floor)

		case isObstructed := <-obstructionEvent:
			elevator.FsmSetObstruction(isObstructed)

		case <-doorTimeoutEvent:
			elevator.FsmOnDoorTimeout()
		}

		time.Sleep(config.INPUT_POLL_RATE)
	}
}

// Transmit the elevator state to the node
func TransmitHRAElevState() {
	for {
		select {
		case <-time.After(config.ELEV_STATE_TRANSMIT_INTERVAL):
			node.ElevatorHRAStatesRx <- elevator.ElevatorState{
				Direction:  elevator.GetDirection(),
				Floor:      elevator.GetFloor(),
				CabRequest: elevator.GetCabRequests(),
				Behavior:   elevator.GetBehavior(),
			}
		}
	}
}
