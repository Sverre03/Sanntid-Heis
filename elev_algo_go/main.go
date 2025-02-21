package main

import (
	"elev/elevator_fsm"
	"elev/elevio"
	"time"
)

func main() {
	buttonEvent := make(chan elevio.ButtonEvent)
	floorEvent := make(chan int)
	doorTimeoutEvent := make(chan bool)
	obstructionEvent := make(chan bool)

	const inputPollRate = 25 * time.Millisecond
	numFloors := 4

	elevio.Init("localhost:15657", numFloors)
	elevator_fsm.InitFSM()

	prevRequestButton := make([][]bool, elevio.NumFloors)
	for i := range prevRequestButton {
		prevRequestButton[i] = make([]bool, elevio.NumButtons)
	}

	// Start polling routines outside the loop
	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)
	go elevio.PollObstructionSwitch(obstructionEvent)
	go elevio.PollTimer(doorTimeoutEvent)

	for {
		select {
		case button := <-buttonEvent:
			elevator_fsm.FsmOnRequestButtonPress(button.Floor, button.Button)

		case floor := <-floorEvent:
			elevator_fsm.FsmOnFloorArrival(floor)

		case isObstructed := <-obstructionEvent:
			elevator_fsm.FsmSetObstruction(isObstructed)

		case <-doorTimeoutEvent:
			elevator_fsm.FsmOnDoorTimeout()
		}

		time.Sleep(inputPollRate)
	}
}
