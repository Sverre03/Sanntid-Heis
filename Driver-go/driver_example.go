package main

import (
	"elev/elevator"
	"fmt"
)

func main() {

	numFloors := 4

	elevator.Init("localhost:15657", numFloors)

	var d elevator.MotorDirection = elevator.MD_Up
	//elevator.SetMotorDirection(d)

	drv_buttons := make(chan elevator.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	go elevator.PollButtons(drv_buttons)
	go elevator.PollFloorSensor(drv_floors)
	go elevator.PollObstructionSwitch(drv_obstr)
	go elevator.PollStopButton(drv_stop)

	for {
		select {
		case a := <-drv_buttons:
			fmt.Printf("%+v\n", a)
			elevator.SetButtonLamp(a.Button, a.Floor, true)

		case a := <-drv_floors:
			fmt.Printf("%+v\n", a)
			if a == numFloors-1 {
				d = elevator.MD_Down
			} else if a == 0 {
				d = elevator.MD_Up
			}
			elevator.SetMotorDirection(d)

		case a := <-drv_obstr:
			fmt.Printf("%+v\n", a)
			if a {
				elevator.SetMotorDirection(elevator.MD_Stop)
			} else {
				elevator.SetMotorDirection(d)
			}

		case a := <-drv_stop:
			fmt.Printf("%+v\n", a)
			for f := 0; f < numFloors; f++ {
				for b := elevator.ButtonType(0); b < 3; b++ {
					elevator.SetButtonLamp(b, f, false)
				}
			}
		}
	}
}
