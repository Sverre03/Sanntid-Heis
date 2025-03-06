package tests

import (
	"elev/Network/network/messages"
	"elev/elevator"
	"elev/node"
	"time"
)

func RunTestNode() {
	Node1 := node.Node(0)
	go node.MasterProgram(Node1)


	Node1.ElevStatesTx <- messages.ElevStates{NodeID: 1, Direction: elevator.MD_Up, Behavior: "idle", Floor: 1, CabRequest: [4]bool{false, true, false, false}}	

	time.Sleep(1 * time.Second)

	newHallReq := messages.NewHallRequest{Floor: 2, HallButton: 0}
	Node1.NewHallReqTx <- newHallReq
}
