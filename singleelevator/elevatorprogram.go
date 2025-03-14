package singleelevator

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
// It manages the elevator state machine, hardware events,
// and communicates with the node.
func ElevatorProgram(
	nodeToElevator chan messages.NodeToElevatorMsg,
	elevatorToNode chan messages.ElevatorToNodeMsg) {

	// Initialize the elevator
	elevator.Init("localhost:15657", config.NUM_FLOORS)
	elevator_fsm.InitFSM()

	// Channels for events
	buttonEvent := make(chan elevator.ButtonEvent)
	floorEvent := make(chan int)
	doorTimeoutEvent := make(chan bool)
	doorStuckEvent := make(chan bool)
	obstructionEvent := make(chan bool)

	// Timers
	doorOpenTimer := timer.NewTimer()  // 3-second door open timer
	doorStuckTimer := timer.NewTimer() // 30-second timer to detect stuck doors

	// Start hardware monitoring routines

	fmt.Println("Starting polling routines")
	go elevator.PollButtons(buttonEvent)
	go elevator.PollFloorSensor(floorEvent)
	go elevator.PollObstructionSwitch(obstructionEvent)
	// start timer
	go func() {
		for range time.Tick(config.INPUT_POLL_INTERVAL) {
			if doorOpenTimer.Active && timer.TimerTimedOut(doorOpenTimer) {
				fmt.Println("Door timer timed out")
				doorTimeoutEvent <- true
			}
		}
	}()

	// Monitor door stuck timeout (30 seconds)
	go func() {
		for range time.Tick(config.INPUT_POLL_INTERVAL) {
			if doorStuckTimer.Active && timer.TimerTimedOut(doorStuckTimer) {
				fmt.Println("Door stuck timer timed out!")
				doorStuckEvent <- true
			}
		}
	}()

	// Transmits the elevator state to the node periodically
	go transmitElevatorState(elevator_fsm.GetElevator(), elevatorToNode)

	for {
		select {
		case button := <-buttonEvent:
			if button.Button == elevator.ButtonCab { // Handle cab calls internally
				elevator_fsm.FsmOnRequestButtonPress(button.Floor, button.Button, &doorOpenTimer)
			} else {
				elevatorToNode <- messages.ElevatorToNodeMsg{
					Type:        messages.MsgHallButtonEvent,
					ButtonEvent: button,
				}
			}

		case msg := <-nodeToElevator:
			switch msg.Type {
			case messages.MsgHallAssignment:
				for floor := 0; floor < config.NUM_FLOORS; floor++ {
					for hallButton := 0; hallButton < 2; hallButton++ {
						if msg.HallAssignments[floor][hallButton] {
							elevator_fsm.FsmOnRequestButtonPress(floor, elevator.ButtonType(hallButton), &doorOpenTimer)
						}
					}
				}

			case messages.MsgRequestDoorState:
				// If door state is requested, send current status
				// isDoorStuck := timer.Active(*doorOpenTimer) && timer.TimerTimeLeft(*doorOpenTimer) > config.DOOR_STUCK_DURATION
				// elevatorToNode <- messages.ElevatorToNodeMsg{
				// 	Type:        messages.MsgDoorStuck,
				// 	IsDoorStuck: isDoorStuck,
				// }
			}

		case floor := <-floorEvent:
			elevator_fsm.FsmOnFloorArrival(floor, &doorOpenTimer, elevatorToNode)

		case isObstructed := <-obstructionEvent:
			elevator_fsm.FsmSetObstruction(isObstructed)

		case <-doorTimeoutEvent:
			if !timer.Active(doorStuckTimer) {
				timer.TimerStart(&doorStuckTimer, config.DOOR_STUCK_DURATION)
			}
			elevator_fsm.FsmOnDoorTimeout(&doorOpenTimer, &doorStuckTimer)

		case <-doorStuckEvent:
			elevatorToNode <- messages.ElevatorToNodeMsg{
				Type:        messages.MsgDoorStuck,
				IsDoorStuck: true,
			}

		case <-time.After(config.INPUT_POLL_INTERVAL):
		}
	}
}

func transmitElevatorState(elev elevator.Elevator, elevatorToNode chan messages.ElevatorToNodeMsg) {
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
}
