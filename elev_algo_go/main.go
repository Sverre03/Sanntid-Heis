package main

import (
	"elevio"
	"fsm"
	"time"
)

func main() {
	buttonEvent := make(chan elevio.ButtonEvent)
	floorEvent := make(chan int)
	doorTimeoutEvent := make(chan bool)
	obstructionEvent := make(chan bool)

	inputPollRateMs := 25
	numFloors := 4

	elevio.Init("localhost:15657", numFloors)
	fsm.InitFSM()

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
			fsm.FsmOnRequestButtonPress(button.Floor, button.Button)

		case floor := <-floorEvent:
			fsm.FsmOnFloorArrival(floor)

		case isObstructed := <-obstructionEvent:
			fsm.FsmSetObstruction(isObstructed)

		case <-doorTimeoutEvent:
			fsm.FsmOnDoorTimeout()
		}

		time.Sleep(time.Duration(inputPollRateMs) * time.Millisecond)
	}
}
