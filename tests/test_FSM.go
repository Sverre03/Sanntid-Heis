package tests

import (
	"elev/node"
)

func RunTestNode() {
	// Node1 := node.Node(1)
	// Node2 := node.Node(2)
	// go node.MasterProgram(Node1)
	// go node.SlaveProgram(Node2)
	Node3 := node.Node(3)
	go node.DisconnectedProgram(Node3)

	// Node2.NewHallReqTx <- messages.NewHallRequest{Floor: 1, HallButton: elevator.BT_HallUp}
	// Node2.ElevStatesTx <- messages.ElevStates{NodeID: 2, Direction: elevator.MD_Up, Floor: 1, CabRequest: [config.NUM_FLOORS]bool{false, false, false, false}, Behavior: "idle"}
}
