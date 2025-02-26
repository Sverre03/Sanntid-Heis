package main

import (
	"elev/elevator"
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

		time.Sleep(config.INPUT_POLL_RATE)
	}
}
