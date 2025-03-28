package node

import (
	"elev/singleelevator"
)

// InactiveProgram runs when the elevator is unavailable or down.
// It waits for the elevator to become operational before transitioning to Disconnected state.
func InactiveProgram(node *NodeData) NodeState {
	var nextNodeState NodeState
MainLoop:
	for {
		select {
		case elevMsg := <-node.ElevatorEventRx:
			// Check if elevator has become operational
			if !elevMsg.ElevIsDown && elevMsg.EventType == singleelevator.ElevStatusUpdateEvent {
				nextNodeState = Disconnected
				break MainLoop
			}

		case <-node.HallAssignmentsRx:
		case <-node.CabRequestInfoRx:
		case <-node.GlobalHallRequestRx:
		case <-node.ConnectionReqRx:
		case <-node.ElevStateUpdatesFromServer:
		case <-node.NetworkEventRx:
		case <-node.MyElevStatesRx:
			// Drain all other channels to prevent blocking
		}
	}
	return nextNodeState
}
