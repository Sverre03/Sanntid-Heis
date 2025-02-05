package main

import (
	"elevio"
	"fmt"
	"fsm"
	"time"
	"timer"
)

func main() {
	fmt.Println("Started!")

	inputPollRateMs := 25
	numFloors := 4

	elevio.Init("localhost:15657", numFloors)

	prevRequestButton := make([][]bool, elevio.NumFloors)
	for i := range prevRequestButton {
		prevRequestButton[i] = make([]bool, elevio.NumButtons)
	}

	prevFloorSensor := -1
	prevObstructed := false

	for {
		// Request button
		for f := 0; f < elevio.NumFloors; f++ {
			for b := 0; b < elevio.NumButtons; b++ {
				v := elevio.GetButton(elevio.ButtonType(b), f)
				if v && v != prevRequestButton[f][b] {
					fsm.FsmOnRequestButtonPress(f, elevio.ButtonType(b))
				}
				prevRequestButton[f][b] = v
			}
		}

		// Obstruction
		isObstructed := elevio.GetObstruction()
		if isObstructed != prevObstructed {
			fsm.FsmSetObstruction(isObstructed)
		}
		prevObstructed = isObstructed

		// Floor sensor
		f := elevio.GetFloor()
		if f != -1 && f != prevFloorSensor {
			fsm.FsmOnFloorArrival(f)
		}
		prevFloorSensor = f

		// Timer
		if timer.TimerTimedOut() {
			timer.TimerStop()
			fsm.FsmOnDoorTimeout()
		}

		time.Sleep(time.Duration(inputPollRateMs) * time.Millisecond)
	}
}
