package elevator

import (
	"elev/util/config"
	"elev/util/timer"
	"fmt"
	"net"
	"sync"
	"time"
)

const _pollRate = 20 * time.Millisecond

var _initialized bool = false

var _mtx sync.Mutex
var _conn net.Conn

type MotorDirection int

const (
	MD_Up   MotorDirection = 1
	MD_Down                = -1
	MD_Stop                = 0
)

type ButtonType int

const (
	BT_HallUp   ButtonType = 0
	BT_HallDown            = 1
	BT_Cab                 = 2
)

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

func Init(addr string, numFloors int) {
	if _initialized {
		fmt.Println("Driver already initialized!")
		return
	}
	_mtx = sync.Mutex{}
	var err error
	_conn, err = net.Dial("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	_initialized = true
}

func SetMotorDirection(dir MotorDirection) {
	write([4]byte{1, byte(dir), 0, 0})
}

func SetButtonLamp(button ButtonType, floor int, value bool) {
	write([4]byte{2, byte(button), byte(floor), toByte(value)})
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
		time.Sleep(_pollRate)
		for floor := 0; floor < config.NUM_FLOORS; floor++ {
			for button := ButtonType(0); button < 3; button++ {
				v := GetButton(button, floor)
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
		time.Sleep(_pollRate)
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
		time.Sleep(_pollRate)
		v := GetStop()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func PollObstructionSwitch(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := GetObstruction()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

// Check if the door has been open for its maximum duration
func PollDoorTimeout(inTimer timer.Timer, receiver chan<- bool) {
	for range time.Tick(config.INPUT_POLL_RATE) {
		if inTimer.Active && timer.TimerTimedOut(inTimer) {
			fmt.Println("Door timer timed out")
			receiver <- true
		}
	}
}

// Check if the door is stuck
func PollDoorStuck(inTimer timer.Timer, receiver chan<- bool) {
	for range time.Tick(50 * time.Millisecond) {
		if inTimer.Active && timer.TimerTimedOut(inTimer) {
			fmt.Println("Door stuck timer timed out!")
			receiver <- true
		}
	}
}

// func PollTimer(inTimer timer.Timer, receiver chan<- bool) {
//     prev := false
//     for {
//         time.Sleep(_pollRate)
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

func GetButton(button ButtonType, floor int) bool {
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

func GetStop() bool {
	a := read([4]byte{8, 0, 0, 0})
	return toBool(a[1])
}

func GetObstruction() bool {
	a := read([4]byte{9, 0, 0, 0})
	return toBool(a[1])
}

func read(in [4]byte) [4]byte {
	_mtx.Lock()
	defer _mtx.Unlock()

	_, err := _conn.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	var out [4]byte
	_, err = _conn.Read(out[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	return out
}

func write(in [4]byte) {
	_mtx.Lock()
	defer _mtx.Unlock()

	_, err := _conn.Write(in[:])
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

func ButtonToString(button ButtonType) string {
	switch button {
	case BT_HallUp:
		return "HallUp"
	case BT_HallDown:
		return "HallDown"
	case BT_Cab:
		return "Cab"
	default:
		return "Unknown"
	}
}

func MotorDirectionToString(dir MotorDirection) string {
	switch dir {
	case MD_Up:
		return "Up"
	case MD_Down:
		return "Down"
	case MD_Stop:
		return "Stop"
	default:
		return "Unknown"
	}
}
