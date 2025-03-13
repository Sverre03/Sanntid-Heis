package single_elevator

import (
	"elev/Network/messages"
	"elev/elevator"
	"elev/elevator_fsm"
	"elev/util/config"
	"elev/util/timer"
	"fmt"
	"time"
)

// ElevatorProgram operates a single elevator
// It manages the elevator state machine, events from hardware,
// and communicates with the hall request assigner.
func ElevatorProgram(
	// ElevatorHallButtonEventTx chan ButtonEvent,
	// ElevatorStateTx chan ElevatorState,
	// ElevatorHallAssignmentRx chan [config.NUM_FLOORS][2]bool,
	// IsDoorStuckCh chan bool,
	// DoorStateRequestCh chan bool) {
	nodeToElevator chan messages.NodeToElevatorMsg,
	elevatorToNode chan messages.ElevatorToNodeMsg) {

	// Initialize the elevator
	elev := elevator.NewElevator()
	elevator.Init("localhost:15657", config.NUM_FLOORS)
	elevator_fsm.InitFSM(&elev)

	// Channels for events
	buttonEvent := make(chan elevator.ButtonEvent)
	floorEvent := make(chan int)
	doorTimeoutEvent := make(chan bool)
	doorStuckEvent := make(chan bool)
	obstructionEvent := make(chan bool)

	doorOpenTimer := timer.NewTimer()  // Used to check if the door is open (if it is not closed after a certain time, 3 seconds)
	doorStuckTimer := timer.NewTimer() // Used to check if the door is stuck (if it is not closed after a certain time, 30 seconds)

	startHardwarePolling(buttonEvent, floorEvent, obstructionEvent)
	startTimerMonitoring(&doorOpenTimer, &doorStuckTimer, doorTimeoutEvent, doorStuckEvent)

	// go transmitElevatorState(&elev, ElevatorStateTx) // Transmits the elevator state to the node periodically
	go func() {
		for range time.Tick(config.ELEV_STATE_TRANSMIT_INTERVAL) {
			elevatorToNode <- messages.ElevatorToNodeMsg{
				Type: messages.MsgElevatorState,
				ElevState: elevator.ElevatorState{
					Behavior:    elev.Behavior,
					Floor:       elev.Floor,
					Direction:   elev.Dir,
					CabRequests: elevator.GetCabRequestsAsElevState(elev),
				},
			}
		}
	}()

	for {
		select {
		case button := <-buttonEvent:
			if (button.Button == elevator.ButtonHallDown) || (button.Button == elevator.ButtonHallUp) {
				elevatorToNode <- messages.ElevatorToNodeMsg{ // Forward the hall call to the node
					Type: messages.MsgHallButtonEvent,
					ButtonEvent: elevator.ButtonEvent{
						Floor:  button.Floor,
						Button: button.Button,
					},
				}
			} else {
				elevator_fsm.FsmOnRequestButtonPress(&elev, button.Floor, button.Button, &doorOpenTimer)
			}
		case msg := <-nodeToElevator:
			switch msg.Type {
			case messages.MsgHallAssignment:
				AssignHallButtons(&elev, msg.HallAssignments, &doorOpenTimer)
			case messages.MsgRequestDoorState:
				// Handle door state request
				// elevatorToNode <- messages.ElevatorToNodeMsg{
				// 	Type:      messages.MsgDoorStuck,
				// 	DoorStuck: true,
				// }
			}

		case floor := <-floorEvent:
			elevator_fsm.FsmOnFloorArrival(&elev, floor, &doorOpenTimer, elevatorToNode)

		case isObstructed := <-obstructionEvent:
			elevator_fsm.FsmSetObstruction(&elev, isObstructed)

		case <-doorTimeoutEvent:
			handleDoorTimeout(&elev, &doorOpenTimer, &doorStuckTimer)

		case <-doorStuckEvent:
			// IsDoorStuckCh <- true
			elevatorToNode <- messages.ElevatorToNodeMsg{
				Type:        messages.MsgDoorStuck,
				IsDoorStuck: true,
			}

		case <-time.After(config.INPUT_POLL_INTERVAL):
		}
	}
}

func startHardwarePolling(buttonEvent chan elevator.ButtonEvent, floorEvent chan int, obstructionEvent chan bool) {
	fmt.Println("Starting polling routines")
	go elevator.PollButtons(buttonEvent)
	go elevator.PollFloorSensor(floorEvent)
	go elevator.PollObstructionSwitch(obstructionEvent)
}

// startTimerMonitoring sets up goroutines to monitor timer events
func startTimerMonitoring(doorOpenTimer *timer.Timer, doorStuckTimer *timer.Timer, doorTimeoutEvent chan bool, doorStuckEvent chan bool) {
	// Monitor door open timeout (3 seconds)
	go func() {
		for range time.Tick(config.INPUT_POLL_INTERVAL) {
			if doorOpenTimer.Active && timer.TimerTimedOut(*doorOpenTimer) {
				fmt.Println("Door timer timed out")
				doorTimeoutEvent <- true
			}
		}
	}()

	// Monitor door stuck timeout (30 seconds)
	go func() {
		for range time.Tick(config.INPUT_POLL_INTERVAL) {
			if doorStuckTimer.Active && timer.TimerTimedOut(*doorStuckTimer) {
				fmt.Println("Door stuck timer timed out!")
				doorStuckEvent <- true
			}
		}
	}()
}

// // Transmit the elevator state to the node
// func transmitElevatorState(elev *Elevator, ElevatorStateRx chan ElevatorState) {
// 	for range time.Tick(config.ELEV_STATE_TRANSMIT_INTERVAL) {
// 		ElevatorStateRx <- ElevatorState{
// 			Behavior:    elev.Behavior,
// 			Floor:       elev.Floor,
// 			Direction:   elev.Dir,
// 			CabRequests: GetCabRequestsAsElevState(*elev),
// 		}
// 	}
// }

func handleButtonEvent(elev *elevator.Elevator, button elevator.ButtonEvent, ElevatorHallButtonEventTx chan elevator.ButtonEvent, doorOpenTimer *timer.Timer) {
	fmt.Printf("Button press detected: Floor %d, Button %s\n",
		button.Floor, button.Button.String())

	if (button.Button == elevator.ButtonHallDown) || (button.Button == elevator.ButtonHallUp) {
		fmt.Printf("Forwarding hall call to node: Floor %d, Button %s\n",
			button.Floor, button.Button.String())
		ElevatorHallButtonEventTx <- elevator.ButtonEvent{ // Forward the hall call to the node
			Floor:  button.Floor,
			Button: button.Button,
		}
	} else {
		elevator_fsm.FsmOnRequestButtonPress(elev, button.Floor, button.Button, doorOpenTimer)
	}
}

func AssignHallButtons(elev *elevator.Elevator, hallButtons [config.NUM_FLOORS][2]bool, doorOpenTimer *timer.Timer) {
	fmt.Printf("Received hall button assignment")
	for floor := 0; floor < config.NUM_FLOORS; floor++ {
		for hallButton := 0; hallButton < 2; hallButton++ {
			elev.Requests[floor][hallButton] = hallButtons[floor][hallButton]
			if elev.Requests[floor][hallButton] {
				elevator_fsm.FsmOnRequestButtonPress(elev, floor, elevator.ButtonType(hallButton), doorOpenTimer)
			}
		}
	}
	elevator.SetAllLights(elev)
}

func handleDoorTimeout(elev *elevator.Elevator, doorOpenTimer *timer.Timer, doorStuckTimer *timer.Timer) {
	fmt.Println("Door timeout event detected")
	if !timer.Active(*doorStuckTimer) {
		timer.TimerStart(doorStuckTimer, config.DOOR_STUCK_DURATION)
	}
	elevator_fsm.FsmOnDoorTimeout(elev, doorOpenTimer, doorStuckTimer)
}
