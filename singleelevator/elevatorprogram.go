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
				elevator_fsm.UpdateHallAssignments(msg.HallAssignments, doorOpenTimer)
				// for floor := range config.NUM_FLOORS {
				// 	for hallButton := range 2 {
				// if msg.HallAssignments[floor][hallButton] { // If the elevator is idle and the button is pressed in the same floor, the door should remain open
				// 	elevator_fsm.OnRequestButtonPress(floor, elevator.ButtonType(hallButton), doorOpenTimer)
				// } else if !msg.HallAssignments[floor][hallButton] && elevator_fsm.GetElevator().Requests[floor][hallButton] { // If hall assignment is removed and redistributed
				// 	// elevator_fsm.RemoveRequest(floor, elevator.ButtonType(hallButton))
				// 	elevator_fsm.RemoveHallAssignment(floor, elevator.ButtonType(hallButton))
				// }

				// }
				// }
				// elevator_fsm.SetHallLights(msg.LightStates)
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
