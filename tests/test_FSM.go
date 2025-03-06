package tests

import (
	"elev/node"
)

func RunTestNode() {
	Node1 := node.Node(1)
	Node2 := node.Node(2)
	go node.MasterProgram(Node1)
	go node.SlaveProgram(Node2)

}
