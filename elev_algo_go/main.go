package main

import (
	"elevio"
	"fmt"
	"fsm"
	"time"
)

func main() {
	buttonEvent := make(chan elevio.ButtonEvent)
	floorEvent := make(chan int)
	// stopEvent  := make(chan bool);
	obstructionEvent := make(chan bool)
	doorTimeout := time.NewTimer(3 * time.Second)
	resetTimeout := make(chan bool)

	fmt.Println("Started!")

	inputPollRateMs := 25
	numFloors := 4

	elevio.Init("localhost:15657", numFloors)

	for f := 0; f < numFloors; f++ {
		for b := 0; b < 3; b++ {
			elevio.SetButtonLamp(elevio.ButtonType(b), f, false)
		}
	}

	prevRequestButton := make([][]bool, elevio.NumFloors)
	for i := range prevRequestButton {
		prevRequestButton[i] = make([]bool, elevio.NumButtons)
	}

	// Start polling routines outside the loop
	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)
	go elevio.PollObstructionSwitch(obstructionEvent)

	for {
		select {
		case button := <-buttonEvent:
			fsm.FsmOnRequestButtonPress(button.Floor, button.Button)

		case floor := <-floorEvent:
			fsm.FsmOnFloorArrival(floor)

		case isObstructed := <-obstructionEvent:
			fsm.FsmSetObstruction(isObstructed)

		case <-doorTimeout.C:
			resetTimeout <- true
			fsm.FsmOnDoorTimeout(resetTimeout)

		case <-resetTimeout:
			doorTimeout.Reset(3 * time.Second)
		}

		time.Sleep(time.Duration(inputPollRateMs) * time.Millisecond)
	}
}
