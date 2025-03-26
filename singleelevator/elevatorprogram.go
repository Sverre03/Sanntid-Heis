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
	HallButtonEvent ElevatorEventType = iota // Receives local hall button presses from node
	DoorStuckEvent                           // Receives the elevator's door state (if it is stuck or not)
)

type ElevatorOrderType int

const (
	HallOrder ElevatorOrderType = iota
	CabOrder
	LightUpdate
)

// ElevatorEvent encapsulates all messages sent from elevator to node
type ElevatorEvent struct {
	EventType   ElevatorEventType
	ButtonEvent elevator.ButtonEvent // For hall button events and completed hall assignments
	DoorIsStuck bool                 // For door stuck status
}

type LightAndAssignmentUpdate struct {
	OrderType       ElevatorOrderType
	HallAssignments [config.NUM_FLOORS][2]bool // For assigning hall calls to the elevator
	CabAssignments  [config.NUM_FLOORS]bool    // For assigning cab calls to the elevator
	LightStates     [config.NUM_FLOORS][2]bool // The new state of the lights
}

func ElevatorProgram(
	portNum string,
	elevatorEventTx chan<- ElevatorEvent,
	elevLightAndAssignmentUpdateRx <-chan LightAndAssignmentUpdate,
	elevatorStatesTx chan<- elevator.ElevatorState) {

	elevator.Init(portNum, config.NUM_FLOORS)
	elevator_fsm.InitFSM()

	buttonEventRx := make(chan elevator.ButtonEvent)
	floorEventRx := make(chan int)
	obstructionEventRx := make(chan bool)

	doorOpenTimer := time.NewTimer(config.DOOR_OPEN_DURATION)   // 3-second timer to detect door timeout
	doorStuckTimer := time.NewTimer(config.DOOR_STUCK_DURATION) // 30-second timer to detect stuck doors
	doorOpenTimer.Stop()
	doorStuckTimer.Stop()

	// Start hardware monitoring routines
	go elevator.PollButtons(buttonEventRx)
	go elevator.PollFloorSensor(floorEventRx)
	go elevator.PollObstructionSwitch(obstructionEventRx)

	elevatorEventTx <- makeDoorStuckMessage(false)

	for {
		select {
		case button := <-buttonEventRx:
			if button.Button == elevator.ButtonCab { // Handle cab calls locally
				elevator_fsm.OnRequestButtonPress(button.Floor, button.Button, doorOpenTimer)
			} else {
				elevatorEventTx <- makeHallReqMessage(button)
			}

		case msg := <-elevLightAndAssignmentUpdateRx:
			switch msg.OrderType {
			case HallOrder:
				elevator_fsm.SetHallLights(msg.LightStates)

				var mergedHallAssignments [config.NUM_FLOORS][2]bool

				// Start with current assignments from the elevator
				for floor := range config.NUM_FLOORS {
					for btn := range 2 {
						mergedHallAssignments[floor][btn] = elevator_fsm.GetElevator().Requests[floor][btn]
					}
				}

				// Add new assignments from the message
				for floor := range config.NUM_FLOORS {
					for btn := range 2 {
						if msg.HallAssignments[floor][btn] {
							mergedHallAssignments[floor][btn] = true
							// This is critical - explicitly notify the FSM about each new button press
							if !elevator_fsm.GetElevator().Requests[floor][btn] {
								btnType := elevator.ButtonType(btn)
								elevator_fsm.OnRequestButtonPress(floor, btnType, doorOpenTimer)
								fmt.Printf("New hall assignment added at floor %d, button %d\n", floor, btn)
							}
						}
					}
				}

				// Only remove assignments that are explicitly not in the message
				// and are present in our current assignments
				for floor := range config.NUM_FLOORS {
					for btn := range 2 {
						if elevator_fsm.GetElevator().Requests[floor][btn] &&
							!msg.HallAssignments[floor][btn] &&
							!msg.LightStates[floor][btn] {
							// This is a hall assignment that should be removed
							mergedHallAssignments[floor][btn] = false
							fmt.Printf("Hall assignment removed at floor %d, button %d\n", floor, btn)
						}
					}
				}

				shouldStop := elevator_fsm.UpdateHallAssignments(mergedHallAssignments)

				if shouldStop {
					elevator_fsm.StopElevator()
				}

				if shouldStop && elevator_fsm.GetElevator().Behavior == elevator.Idle {
					elevator_fsm.ResumeElevator()
				}

				fmt.Printf("Hall assignments received: %v\n", msg.HallAssignments)
				var localHallAssignments [config.NUM_FLOORS][2]bool
				for floor := range config.NUM_FLOORS {
					for hallButton := range 2 {
						if elevator_fsm.GetElevator().Requests[floor][hallButton] {
							localHallAssignments[floor][hallButton] = true
						}
					}
				}
				fmt.Printf("My local hall assignments: %v\n", localHallAssignments)
				fmt.Printf("Light states            : %v\n", msg.LightStates)
				fmt.Printf("My elevator hall lights: %v\n\n", elevator_fsm.GetElevator().HallLightStates)

			case CabOrder:
				for floor := range config.NUM_FLOORS {
					if msg.CabAssignments[floor] {
						elevator_fsm.OnRequestButtonPress(floor, elevator.ButtonCab, doorOpenTimer)
					}
				}
			case LightUpdate:
				elevator_fsm.SetHallLights(msg.LightStates)
				fmt.Printf("Light states            : %v\n", msg.LightStates)
				fmt.Printf("My elevator hall lights: %v\n\n", elevator_fsm.GetElevator().HallLightStates)
			}

		case floor := <-floorEventRx:
			elevator_fsm.OnFloorArrival(floor, doorOpenTimer)

		case isObstructed := <-obstructionEventRx:
			fmt.Printf("Obstruction detected: %v\n", isObstructed)
			elevator_fsm.SetObstruction(isObstructed)
			if !isObstructed {
				// Stop the door stuck timer if the obstruction is cleared
				doorStuckTimer.Stop()
				elevatorEventTx <- makeDoorStuckMessage(false)
			}

		case <-doorOpenTimer.C:
			// Start the door stuck timer, which is stopped by OnDoorTimeout if the doors are able to close
			elevator_fsm.OnDoorTimeout(doorOpenTimer, doorStuckTimer)

		case <-doorStuckTimer.C:
			fmt.Println("Door stuck timer timed out")
			elevatorEventTx <- makeDoorStuckMessage(true)

		case <-time.Tick(config.ELEV_STATE_TRANSMIT_INTERVAL):
			elev := elevator_fsm.GetElevator()
			var localHallAssignments [config.NUM_FLOORS][2]bool
			for floor := range config.NUM_FLOORS {
				for button := range 2 {
					localHallAssignments[floor][button] = elev.Requests[floor][button]
				}
			}
			// elevator.PrintElevator(elevator_fsm.GetElevator())
			elevatorStatesTx <- elevator.ElevatorState{
				Behavior:          elev.Behavior,
				Floor:             elev.Floor,
				Direction:         elev.Dir,
				CabRequests:       elevator.GetCabRequestsAsElevState(elev),
				MyHallAssignments: localHallAssignments,
			}
		}
	}
}

func makeDoorStuckMessage(isDoorStuck bool) ElevatorEvent {
	return ElevatorEvent{EventType: DoorStuckEvent,
		DoorIsStuck: isDoorStuck,
	}
}

func makeHallReqMessage(buttonEvent elevator.ButtonEvent) ElevatorEvent {
	return ElevatorEvent{EventType: HallButtonEvent,
		ButtonEvent: buttonEvent,
	}
}
