package singleelevator

import (
	"elev/elevator"
	"elev/elevator_fsm"
	"elev/util/config"
	"elev/util/timer"
	"fmt"
	"time"
)

type ElevatorEventType int

const (
	HallButtonEvent                  ElevatorEventType = iota // Receives local hall button presses from node
	LocalHallAssignmentCompleteEvent                          // Receives completed hall assignments
	DoorStuckEvent                                            // Receives the elevator's door state (if it is stuck or not)
)

// ElevatorEventMsg encapsulates all messages sent from elevator to node
type ElevatorEvent struct {
	EventType   ElevatorEventType
	ButtonEvent elevator.ButtonEvent // For hall button events and completed hall assignments
	IsDoorStuck bool                 // For door stuck status
}

// NodeToElevatorMsg encapsulates all messages sent from node to elevator
type LightAndHallAssignmentUpdate struct {
	HallAssignments      [config.NUM_FLOORS][2]bool // For assigning hall calls to the elevator
	HallAssignmentAreNew bool                       // are the hall assignments new
	LightStates          [config.NUM_FLOORS][2]bool // The new state of the lights
}

// ElevatorProgram operates a single elevator
// It manages the elevator state machine, hardware events,
// and communicates with the node.
func ElevatorProgram(
	elevatorEventTx chan<- ElevatorEvent,
	elevPanelUpdateRx <-chan LightAndHallAssignmentUpdate,
	elevatorStatesTx chan<- elevator.ElevatorState,
) {

	// Initialize the elevator
	elevator.Init("localhost:15657", config.NUM_FLOORS)
	elevator_fsm.InitFSM()

	// Channels for events
	buttonEventRx := make(chan elevator.ButtonEvent)
	floorEventRx := make(chan int)
	doorTimeoutEventRx := make(chan bool)
	doorStuckEventRx := make(chan bool)
	obstructionEventRx := make(chan bool)

	// Timers
	doorOpenTimer := timer.NewTimer()  // 3-second door open timer
	doorStuckTimer := timer.NewTimer() // 30-second timer to detect stuck doors

	// Start hardware monitoring routines
	fmt.Println("Starting polling routines")
	go elevator.PollButtons(buttonEventRx)
	go elevator.PollFloorSensor(floorEventRx)
	go elevator.PollObstructionSwitch(obstructionEventRx)

	// start timer
	go func() {
		for range time.Tick(config.TIMEOUT_TIMER_POLL_INTERVALL) {
			if doorOpenTimer.Active && timer.TimerTimedOut(doorOpenTimer) {
				fmt.Println("Door timer timed out")
				doorTimeoutEventRx <- true
			}
		}
	}()

	// Monitor door stuck timeout (30 seconds)
	go func() {
		for range time.Tick(config.TIMEOUT_TIMER_POLL_INTERVALL) {
			if doorStuckTimer.Active && timer.TimerTimedOut(doorStuckTimer) {
				fmt.Println("Door stuck timer timed out!")
				doorStuckEventRx <- true
			}
		}
	}()

	// Transmits the elevator state to the node periodically
	go transmitElevatorState(elevatorStatesTx)

	for {
		select {
		case button := <-buttonEventRx:
			if button.Button == elevator.ButtonCab { // Handle cab calls internally
				elevator_fsm.FsmOnRequestButtonPress(button.Floor, button.Button, &doorOpenTimer)
			} else {
				elevatorEventTx <- makeHallBtnEventMessage(button)
			}

		case msg := <-elevPanelUpdateRx:
			if msg.HallAssignmentAreNew {
				for floor := 0; floor < config.NUM_FLOORS; floor++ {
					for hallButton := 0; hallButton < 2; hallButton++ {
						if msg.HallAssignments[floor][hallButton] {
							elevator_fsm.FsmOnRequestButtonPress(floor, elevator.ButtonType(hallButton), &doorOpenTimer)
						}
					}
				}
			}

		case floor := <-floorEventRx:
			clearedButtonEvents := elevator_fsm.FsmOnFloorArrival(floor, &doorOpenTimer)

			// loop through and send the button events!
			for _, buttonEvent := range clearedButtonEvents {
				if buttonEvent.Button != elevator.ButtonCab {
					elevatorEventTx <- makeHallBtnEventMessage(buttonEvent)
				}
			}

		case isObstructed := <-obstructionEventRx:
			elevator_fsm.FsmSetObstruction(isObstructed)

		case <-doorTimeoutEventRx:
			// start the door stuck timer. It is stopped by FsmOnDoorTimeout if the doors are able to close
			if !timer.Active(doorStuckTimer) {
				timer.TimerStart(&doorStuckTimer, config.DOOR_STUCK_DURATION)
			}
			elevator_fsm.FsmOnDoorTimeout(&doorOpenTimer, &doorStuckTimer)

		case <-doorStuckEventRx:
			elevatorEventTx <- makeDoorStuckMessage(true)
		}
	}
}

func transmitElevatorState(elevatorToNode chan<- elevator.ElevatorState) {

	for range time.Tick(config.ELEV_STATE_TRANSMIT_INTERVAL) {
		// call getelevator
		elev := elevator_fsm.GetElevator()

		elevatorToNode <- elevator.ElevatorState{
			Behavior:    elev.Behavior,
			Floor:       elev.Floor,
			Direction:   elev.Dir,
			CabRequests: elevator.GetCabRequestsAsElevState(elev),
		}
	}
}

func makeHallBtnEventMessage(buttonEvent elevator.ButtonEvent) ElevatorEvent {
	return ElevatorEvent{EventType: HallButtonEvent,
		ButtonEvent: buttonEvent, IsDoorStuck: false}
}

func makeDoorStuckMessage(isDoorStuck bool) ElevatorEvent {
	return ElevatorEvent{EventType: DoorStuckEvent,
		IsDoorStuck: isDoorStuck,
	}
}
