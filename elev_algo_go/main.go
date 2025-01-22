package main

import (
    "fmt"
    "time"
)

func main() {
    fmt.Println("Started!")

    inputPollRateMs := 25

    input := elevioGetInputDevice()

    if input.FloorSensor() == -1 {
        fsmOnInitBetweenFloors()
    }

    prevRequestButton := make([][]int, N_FLOORS)
    for i := range prevRequestButton {
        prevRequestButton[i] = make([]int, N_BUTTONS)
    }

    prevFloorSensor := -1

    for {
        // Request button
        for f := 0; f < N_FLOORS; f++ {
            for b := 0; b < N_BUTTONS; b++ {
                v := input.RequestButton(f, b)
                if v != 0 && v != prevRequestButton[f][b] {
                    fsmOnRequestButtonPress(f, b)
                }
                prevRequestButton[f][b] = v
            }
        }

        // Floor sensor
        f := input.FloorSensor()
        if f != -1 && f != prevFloorSensor {
            fsmOnFloorArrival(f)
        }
        prevFloorSensor = f

        // Timer
        if timerTimedOut() {
            timerStop()
            fsmOnDoorTimeout()
        }

        time.Sleep(time.Duration(inputPollRateMs) * time.Millisecond)
    }
}
