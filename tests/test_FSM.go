package tests

import (
	"elev/node"
)

func RunTestNode() {
	Node1 := node.MakeNode(1)
	go node.SlaveProgram(Node1)

	// Node1.NodeElevStatesTx <- messages.ElevStates{NodeID: 1, Direction: elevator.DirectionUp, Behavior: "idle", Floor: 1, CabRequest: [4]bool{false, true, false, false}}

	// time.Sleep(1 * time.Second)

	// newHallReq := messages.NewHallRequest{Floor: 2, HallButton: 0}
	// Node1.NewHallReqTx <- newHallReq
}
