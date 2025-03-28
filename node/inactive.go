package node

import (
	"elev/singleelevator"
)

// InactiveProgram runs when the elevator is unavailable or down.
// It waits for the elevator to become operational before transitioning to Disconnected state.
func InactiveProgram(node *NodeData) NodeState {
	var nextNodeState NodeState
ForLoop:
	for {
		select {

		case elevMsg := <-node.ElevatorEventRx:

			// If the elevator is operational, go to Disconnected state
			if !elevMsg.ElevIsDown && elevMsg.EventType == singleelevator.ElevStatusUpdateEvent {
				nextNodeState = Disconnected
				break ForLoop
			}

		case <-node.HallAssignmentsRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
		case <-node.ConnectionReqRx:
		case <-node.ElevStateUpdatesFromServer:
		case <-node.NetworkEventRx:
		case <-node.MyElevStatesRx:
			// Read these to prevent blocking
		}
	}
	return nextNodeState
}
