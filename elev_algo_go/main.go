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

	elevio.Init("localhost:15657", 4)

	prevRequestButton := make([][]bool, elevio.NumFloors)
	for i := range prevRequestButton {
		prevRequestButton[i] = make([]bool, elevio.NumButtons)
	}

	prevFloorSensor := -1

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
