package singleelevator

import (
	"elev/config"
	"elev/elevator"
	"elev/elevator_fsm"
	"fmt"
	"time"
)

type ElevatorEventType int

const (
	HallButtonEvent                  ElevatorEventType = iota // Receives local hall button presses from node
	LocalHallAssignmentCompleteEvent                          // Receives completed hall assignments
	DoorStuckEvent                                            // Receives the elevator's door state (if it is stuck or not)
)

type ElevatorOrderType int

const (
	HallOrder ElevatorOrderType = iota
	CabOrder
	LightUpdate
)

// ElevatorEventMsg encapsulates all messages sent from elevator to node
type ElevatorEvent struct {
	EventType   ElevatorEventType
	ButtonEvent elevator.ButtonEvent // For hall button events and completed hall assignments
	DoorIsStuck bool                 // For door stuck status
}

// NodeToElevatorMsg encapsulates all messages sent from node to elevator
type LightAndAssignmentUpdate struct {
	OrderType       ElevatorOrderType
	HallAssignments [config.NUM_FLOORS][2]bool // For assigning hall calls to the elevator
	CabAssignments  [config.NUM_FLOORS]bool    // For assigning cab calls to the elevator
	LightStates     [config.NUM_FLOORS][2]bool // The new state of the lights
}

// ElevatorProgram operates a single elevator
// It manages the elevator state machine, hardware events,
// and communicates with the node.
func ElevatorProgram(
	portNum string,
	elevatorEventTx chan<- ElevatorEvent,
	elevLightAndAssignmentUpdateRx <-chan LightAndAssignmentUpdate,
	elevatorStatesTx chan<- elevator.ElevatorState) {

	elevator.Init(portNum, config.NUM_FLOORS) // "localhost:15657"
	elevator_fsm.InitFSM()

	// Channels for events
	buttonEventRx := make(chan elevator.ButtonEvent)
	floorEventRx := make(chan int)
	obstructionEventRx := make(chan bool)

	// Timers
	doorStuckTimerActive := false

	doorOpenTimer := time.NewTimer(config.DOOR_OPEN_DURATION)   // 3-second timer to detect door timeout
	doorStuckTimer := time.NewTimer(config.DOOR_STUCK_DURATION) // 30-second timer to detect stuck doors
	doorOpenTimer.Stop()
	doorStuckTimer.Stop()

	// Start hardware monitoring routines
	fmt.Println("Starting polling routines")
	go elevator.PollButtons(buttonEventRx)
	go elevator.PollFloorSensor(floorEventRx)
	go elevator.PollObstructionSwitch(obstructionEventRx)

	// Transmits the elevator state to the node periodically
	go transmitElevatorState(elevatorStatesTx)

	// Check if door is stuck
	elevatorEventTx <- makeDoorStuckMessage(false)

	for {
		select {
		case button := <-buttonEventRx:
			if button.Button == elevator.ButtonCab { // Handle cab calls internally
				elevator_fsm.OnRequestButtonPress(button.Floor, button.Button, doorOpenTimer)
			} else {
				elevatorEventTx <- makeHallButtonEventMessage(button)
			}

		case msg := <-elevLightAndAssignmentUpdateRx:
			switch msg.OrderType {
			case HallOrder:
				for floor := 0; floor < config.NUM_FLOORS; floor++ {
					for hallButton := 0; hallButton < 2; hallButton++ {
						if msg.HallAssignments[floor][hallButton] { // If the elevator is idle and the button is pressed in the same floor, the door should remain open
							clearedEvents := elevator_fsm.OnRequestButtonPress(floor, elevator.ButtonType(hallButton), doorOpenTimer)
							for _, buttonEvent := range clearedEvents {
								if buttonEvent.Button != elevator.ButtonCab && buttonEvent.Floor == floor {
									elevatorEventTx <- makeHallAssignmentCompleteEventMessage(buttonEvent)
								}
							}
						} else if !msg.HallAssignments[floor][hallButton] && elevator_fsm.GetElevator().Requests[floor][hallButton] {
							elevator_fsm.RemoveRequest(floor, elevator.ButtonType(hallButton))
						}

					}
				}
			case CabOrder:
				for floor := 0; floor < config.NUM_FLOORS; floor++ {
					if msg.CabAssignments[floor] {
						elevator_fsm.OnRequestButtonPress(floor, elevator.ButtonCab, doorOpenTimer)
					}
				}
			case LightUpdate:
				elevator_fsm.SetHallLights(msg.LightStates)
			}

		case floor := <-floorEventRx:
			clearedButtonEvents := elevator_fsm.OnFloorArrival(floor, doorOpenTimer)

			// loop through and send the button events!
			for _, buttonEvent := range clearedButtonEvents {
				fmt.Printf("Button event: %v\n", buttonEvent)
				if buttonEvent.Button != elevator.ButtonCab {
					elevatorEventTx <- makeHallAssignmentCompleteEventMessage(buttonEvent)
				}
			}

		case isObstructed := <-obstructionEventRx:
			elevator_fsm.SetObstruction(isObstructed)

		case <-doorOpenTimer.C:
			// Start the door stuck timer, which is stopped by OnDoorTimeout if the doors are able to close

			if !doorStuckTimerActive {
				doorStuckTimer.Reset(config.DOOR_STUCK_DURATION)
				doorStuckTimerActive = true
			}
			elevator_fsm.OnDoorTimeout(doorOpenTimer, doorStuckTimer)

		case <-doorStuckTimer.C:
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

func makeHallButtonEventMessage(buttonEvent elevator.ButtonEvent) ElevatorEvent {
	return ElevatorEvent{EventType: HallButtonEvent,
		ButtonEvent: buttonEvent, DoorIsStuck: false}
}

func makeHallAssignmentCompleteEventMessage(buttonEvent elevator.ButtonEvent) ElevatorEvent {
	return ElevatorEvent{EventType: LocalHallAssignmentCompleteEvent,
		ButtonEvent: buttonEvent, DoorIsStuck: false}
}

func makeDoorStuckMessage(isDoorStuck bool) ElevatorEvent {
	return ElevatorEvent{EventType: DoorStuckEvent,
		DoorIsStuck: isDoorStuck,
	}
}
