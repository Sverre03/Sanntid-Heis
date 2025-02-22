package main

import (
	"elev/elevator"
	"time"
)

func main() {
	buttonEvent := make(chan elevator.ButtonEvent)
	floorEvent := make(chan int)
	doorTimeoutEvent := make(chan bool)
	obstructionEvent := make(chan bool)

	const inputPollRate = 25 * time.Millisecond
	numFloors := 4

	elevator.Init("localhost:15657", numFloors)
	elevator.InitFSM()

	prevRequestButton := make([][]bool, elevator.NumFloors)
	for i := range prevRequestButton {
		prevRequestButton[i] = make([]bool, elevator.NumButtons)
	}

	// Start polling routines outside the loop
	go elevator.PollButtons(buttonEvent)
	go elevator.PollFloorSensor(floorEvent)
	go elevator.PollObstructionSwitch(obstructionEvent)
	go elevator.PollTimer(doorTimeoutEvent)

	for {
		select {
		case button := <-buttonEvent:
			elevator.FsmOnRequestButtonPress(button.Floor, button.Button)

		case floor := <-floorEvent:
			elevator.FsmOnFloorArrival(floor)

		case isObstructed := <-obstructionEvent:
			elevator.FsmSetObstruction(isObstructed)

		case <-doorTimeoutEvent:
			elevator.FsmOnDoorTimeout()
		}

		time.Sleep(inputPollRate)
	}
}
