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
	HallButtonEvent       ElevatorEventType = iota // Receives local hall button presses from node
	ElevStatusUpdateEvent                          // Receives the elevator's door state (if it is stuck or not)
)

type ElevatorOrderType int

const (
	HallAssignment ElevatorOrderType = iota
	CabAssignment
	LightUpdate
)

// ElevatorEvent encapsulates all messages sent from elevator to node
type ElevatorEvent struct {
	EventType   ElevatorEventType
	ButtonEvent elevator.ButtonEvent // For hall button events and completed hall assignments
	ElevIsDown  bool                 // For elevator status
}

type LightAndAssignmentUpdate struct {
	OrderType                  ElevatorOrderType
	HallAssignments            [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool // For assigning hall calls to the elevator
	CabAssignments             [config.NUM_FLOORS]bool                          // For assigning cab calls to the elevator
	LightStates                [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool // The new state of the lights
	HallAssignmentCounterValue int
}

func ElevatorProgram(
	portNum string,
	elevLightAndAssignmentUpdateRx <-chan LightAndAssignmentUpdate,
	elevatorEventTx chan<- ElevatorEvent,
	elevatorStatesTx chan<- elevator.ElevatorStateReport) {

	elevator.Init(portNum, config.NUM_FLOORS)
	elevator_fsm.InitFSM()

	buttonEventRx := make(chan elevator.ButtonEvent)
	floorEventRx := make(chan int)
	obstructionEventRx := make(chan bool)

	doorOpenTimer := time.NewTimer(config.DOOR_OPEN_DURATION)   // 3-second timer to detect door timeout
	doorStuckTimer := time.NewTimer(config.DOOR_STUCK_DURATION) // 30-second timer to detect stuck doors
	stuckBetweenFloorsTimer := time.NewTimer(config.ELEV_STUCK_TIMEOUT)

	doorOpenTimer.Stop()
	doorStuckTimer.Stop()
	stuckBetweenFloorsTimer.Stop()

	lastFloorChange := time.Now()
	recoveryAttempts := 0
	maxRecoveryAttempts := 3
	HallAssignmentCounterValue := -1

	// Start hardware monitoring routines
	go elevator.PollButtons(buttonEventRx)
	go elevator.PollFloorSensor(floorEventRx)
	go elevator.PollObstructionSwitch(obstructionEventRx)

	startStuckMonitoring := func() {
		if elevator_fsm.GetElevator().Behavior == elevator.Moving {
			stuckBetweenFloorsTimer.Reset(config.ELEV_STUCK_TIMEOUT)
			lastFloorChange = time.Now()
		}
	}

	for {
		select {
		case button := <-buttonEventRx:
			if button.Button == elevator.ButtonCab { // Handle cab calls locally
				elevator_fsm.OnRequestButtonPress(button.Floor, button.Button, doorOpenTimer)
				startStuckMonitoring()
			} else {
				elevatorEventTx <- makeHallReqMessage(button)
			}

		case msg := <-elevLightAndAssignmentUpdateRx:
			switch msg.OrderType {
			case HallAssignment:

				elevator_fsm.UpdateHallLightStates(msg.LightStates)

				HallAssignmentCounterValue = msg.HallAssignmentCounterValue

				shouldStop := elevator_fsm.RemoveInvalidHallAssignments(msg.HallAssignments)

				addedHallAssignments := addNewHallAssignments(msg.HallAssignments)
				// Loop all added hall assignments and add them to elevator requests
				for floor, btn := range addedHallAssignments {
					for btn, isActive := range btn {
						if isActive {
							btnType := elevator.ButtonType(btn)
							elevator_fsm.OnRequestButtonPress(floor, btnType, doorOpenTimer)
						}
					}
				}

				currentElevator := elevator_fsm.GetElevator()

				if shouldStop && currentElevator.Behavior == elevator.Moving {
					elevator_fsm.StopElevator()

					currentElevator = elevator_fsm.GetElevator()

					if hasAssignments(currentElevator.Requests) {
						elevator_fsm.ResumeElevator()
					} else {
						elevator_fsm.RecoverFromStuckBetweenFloors()
					}
				}

			case CabAssignment:
				// Add my own cab requests to elevator
				for floor := range config.NUM_FLOORS {
					if msg.CabAssignments[floor] {
						elevator_fsm.OnRequestButtonPress(floor, elevator.ButtonCab, doorOpenTimer)
					}
				}
			case LightUpdate:
				elevator_fsm.UpdateHallLightStates(msg.LightStates)
			}

			startStuckMonitoring()

		case floor := <-floorEventRx:
			elevatorEventTx <- makeElevatorIsDownMessage(false) // Report elevator as up when we reach a floor
			stuckBetweenFloorsTimer.Stop()
			recoveryAttempts = 0 // Reset recovery attempts when we reach a floor
			lastFloorChange = time.Now()

			elevator_fsm.OnFloorArrival(floor, doorOpenTimer)
			startStuckMonitoring()

		case isObstructed := <-obstructionEventRx:
			elevator_fsm.SetObstruction(isObstructed)
			if !isObstructed {
				// Stop the door stuck timer if the obstruction is cleared
				doorStuckTimer.Stop()
				elevator_fsm.SetElevDoorStuckTimerActive(false)
				elevatorEventTx <- makeElevatorIsDownMessage(false)
			}

		case <-doorOpenTimer.C:
			elevator_fsm.OnDoorTimeout(doorOpenTimer, doorStuckTimer)

		case <-doorStuckTimer.C:
			elevatorEventTx <- makeElevatorIsDownMessage(true)

		case <-stuckBetweenFloorsTimer.C:
			if recoveryAttempts < maxRecoveryAttempts {
				fmt.Printf("Attempting recovery (attempt %d of %d)...\n", recoveryAttempts+1, maxRecoveryAttempts)

				elevator_fsm.RecoverFromStuckBetweenFloors()
				recoveryAttempts++

				// Start monitoring again
				startStuckMonitoring()
			} else {
				fmt.Printf("Failed to recover after %d attempts - reporting elevator as down\n", maxRecoveryAttempts)

				elevatorEventTx <- makeElevatorIsDownMessage(true)
			}
		case <-time.Tick(config.ELEV_STUCK_POLL_INTERVAL):
			elev := elevator_fsm.GetElevator()

			// If we're supposed to be moving but haven't changed floors in too long
			if elev.Behavior == elevator.Moving && time.Since(lastFloorChange) > config.ELEV_STUCK_TIMEOUT {
				fmt.Println("Detected elevator not moving between floors (timeout)")
				stuckBetweenFloorsTimer.Stop()   // Stop current timer
				stuckBetweenFloorsTimer.Reset(0) // Trigger immediately
			}

			// If we're in StoppedBetweenFloors but have assignments, attempt to resume operation
			if elev.Behavior == elevator.StoppedBetweenFloors && hasAssignments(elev.Requests) {
				fmt.Println("Detected elevator stopped between floors with pending requests")
				fmt.Println("Attempting to resume operation...")
				elevator_fsm.ResumeElevator()
				startStuckMonitoring()

				// If it's still not moving after resume attempt, something is wrong
				if elevator_fsm.GetElevator().Behavior != elevator.Moving {
					fmt.Println("Failed to resume operation - attempting emergency recovery")
					stuckBetweenFloorsTimer.Stop()   // Stop current timer
					stuckBetweenFloorsTimer.Reset(0) // Trigger immediately
				} else {
					startStuckMonitoring()
				}
			}

		case <-time.Tick(config.ELEV_STATE_TRANSMIT_INTERVAL):
			// Periodically send elevator state report to node
			elev := elevator_fsm.GetElevator()

			elevatorStatesTx <- elevator.ElevatorStateReport{
				Behavior:          elev.Behavior,
				Floor:             elev.Floor,
				Direction:         elev.Dir,
				CabRequests:       getCabRequests(elev),
				MyHallAssignments: getElevatorHallAssignments(elev),
				HACounterVersion:  HallAssignmentCounterValue,
			}
		}
	}
}

func makeElevatorIsDownMessage(ElevIsDown bool) ElevatorEvent {
	return ElevatorEvent{EventType: ElevStatusUpdateEvent,
		ElevIsDown: ElevIsDown,
	}
}

func makeHallReqMessage(buttonEvent elevator.ButtonEvent) ElevatorEvent {
	return ElevatorEvent{EventType: HallButtonEvent,
		ButtonEvent: buttonEvent,
	}
}

func getCabRequests(elev elevator.Elevator) [config.NUM_FLOORS]bool {
	var cabRequests [config.NUM_FLOORS]bool
	for floor := range config.NUM_FLOORS {
		cabRequests[floor] = elev.Requests[floor][elevator.ButtonCab]
	}
	return cabRequests
}

func getElevatorHallAssignments(elev elevator.Elevator) [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool {
	var elevatorHallAssignments [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool
	for floor := range config.NUM_FLOORS {
		for button := range config.NUM_HALL_BUTTONS {
			elevatorHallAssignments[floor][button] = elev.Requests[floor][button]
		}
	}
	return elevatorHallAssignments
}

func hasAssignments(requests [config.NUM_FLOORS][config.NUM_BUTTONS]bool) bool {
	for floor := range config.NUM_FLOORS {
		for btn := range config.NUM_BUTTONS {
			if requests[floor][btn] {
				return true
			}
		}
	}
	return false
}

func addNewHallAssignments(newHallAssignments [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool) [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool {
	var addedHallAssignments [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool
	for floor := range config.NUM_FLOORS {
		for btn := range config.NUM_HALL_BUTTONS {
			if newHallAssignments[floor][btn] && !elevator_fsm.GetElevator().Requests[floor][btn] { // If this assignment dont already exist in elevator requests add it as new hall assignment
				addedHallAssignments[floor][btn] = true
			}
		}
	}
	return addedHallAssignments
}
