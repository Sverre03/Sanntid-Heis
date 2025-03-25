package elevator

import (
	"elev/config"
	"elev/util/timer"
	"fmt"
	"net"
	"sync"
	"time"
)

const pollInterval = 20 * time.Millisecond

var driverIsInitialized bool = false
var driverMutex sync.Mutex
var serverConnection net.Conn

type MotorDirection int

const (
	DirectionUp   MotorDirection = 1
	DirectionDown                = -1
	DirectionStop                = 0
)

type ButtonType int

const (
	ButtonHallUp   ButtonType = 0
	ButtonHallDown            = 1
	ButtonCab                 = 2
)

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

func Init(addr string, numFloors int) {
	if driverIsInitialized {
		fmt.Println("Driver already initialized!")
		return
	}
	driverMutex = sync.Mutex{}
	var err error
	serverConnection, err = net.Dial("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	driverIsInitialized = true
}

func SetMotorDirection(dir MotorDirection) {
	write([4]byte{1, byte(dir), 0, 0})
}

func SetButtonLamp(button ButtonType, floor int, value bool) {
	write([4]byte{2, byte(button), byte(floor), toByte(value)})
}

func SetAllLights(elev *Elevator) {
	for floor := range config.NUM_FLOORS {
		SetButtonLamp(ButtonCab, floor, elev.Requests[floor][ButtonCab])

		for i := range config.NUM_BUTTONS-1 {
			SetButtonLamp(ButtonType(i), floor, elev.HallLightStates[floor][ButtonType(i)])
		}
	}
}

func SetFloorIndicator(floor int) {
	write([4]byte{3, byte(floor), 0, 0})
}

func SetDoorOpenLamp(value bool) {
	write([4]byte{4, toByte(value), 0, 0})
}

func SetStopLamp(value bool) {
	write([4]byte{5, toByte(value), 0, 0})
}

func PollButtons(receiver chan<- ButtonEvent) {
	prev := make([][3]bool, config.NUM_FLOORS)
	for {
		time.Sleep(pollInterval)
		for floor := range config.NUM_FLOORS {
			for button := ButtonType(0); button < 3; button++ {
				v := ButtonIsPressed(button, floor)
				if v != prev[floor][button] && v {
					receiver <- ButtonEvent{floor, ButtonType(button)}
				}
				prev[floor][button] = v
			}
		}
	}
}

func PollFloorSensor(receiver chan<- int) {
	prev := -1
	for {
		time.Sleep(pollInterval)
		v := GetFloor()
		if v != prev && v != -1 {
			receiver <- v
		}
		prev = v
	}
}

func PollStopButton(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(pollInterval)
		v := StopIsPressed()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func PollObstructionSwitch(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(pollInterval)
		v := Obstructed()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

// Check if the door has been open for its maximum duration
func PollDoorTimeout(inTimer timer.Timer, receiver chan<- bool) {
	for range time.Tick(config.INPUT_POLL_INTERVAL) {
		if inTimer.Active && timer.TimerTimedOut(inTimer) {
			fmt.Println("Door timer timed out")
			receiver <- true
		}
	}
}

// Check if the door is stuck
func PollDoorStuck(inTimer timer.Timer, receiver chan<- bool) {
	for range time.Tick(config.INPUT_POLL_INTERVAL) {
		if inTimer.Active && timer.TimerTimedOut(inTimer) {
			fmt.Println("Door stuck timer timed out!")
			receiver <- true
		}
	}
}

// func PollTimer(inTimer timer.Timer, receiver chan<- bool) {
//     prev := false
//     for {
//         time.Sleep(pollInterval)
//         // IMPORTANT FIX: Get current timer status instead of keeping a local reference
//         currentTimerValue := timer.TimerTimedOut(inTimer)

//         // Only send when transitioning from false to true
//         if currentTimerValue && !prev {
//             fmt.Printf("Timer timed out! Active=%v, EndTime=%v\n",
//                 inTimer.Active, inTimer.EndTime.Format("15:04:05.000"))
//             receiver <- true
//         }
//         prev = currentTimerValue
//     }
// }

func ButtonIsPressed(button ButtonType, floor int) bool {
	a := read([4]byte{6, byte(button), byte(floor), 0})
	return toBool(a[1])
}

func GetFloor() int {
	a := read([4]byte{7, 0, 0, 0})
	if a[1] != 0 {
		return int(a[2])
	} else {
		return -1
	}
}

func StopIsPressed() bool {
	a := read([4]byte{8, 0, 0, 0})
	return toBool(a[1])
}

func Obstructed() bool {
	a := read([4]byte{9, 0, 0, 0})
	return toBool(a[1])
}

func read(in [4]byte) [4]byte {
	driverMutex.Lock()
	defer driverMutex.Unlock()

	_, err := serverConnection.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	var out [4]byte
	_, err = serverConnection.Read(out[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	return out
}

func write(in [4]byte) {
	driverMutex.Lock()
	defer driverMutex.Unlock()

	_, err := serverConnection.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}
}

func toByte(a bool) byte {
	var b byte = 0
	if a {
		b = 1
	}
	return b
}

func toBool(a byte) bool {
	var b bool = false
	if a != 0 {
		b = true
	}
	return b
}

// func (button ButtonType) String() string {
// 	switch button {
// 	case ButtonHallUp:
// 		return "HallUp"
// 	case ButtonHallDown:
// 		return "HallDown"
// 	case ButtonCab:
// 		return "Cab"
// 	default:
// 		return "Unknown"
// 	}
// }

// func (dir MotorDirection) String() string {
// 	switch dir {
// 	case DirectionUp:
// 		return "Up"
// 	case DirectionDown:
// 		return "Down"
// 	case DirectionStop:
// 		return "Stop"
// 	default:
// 		return "Unknown"
// 	}
// }
