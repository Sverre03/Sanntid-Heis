package testing

import "elev/node"

func CreateTestNode() {
	testNode := node.Node(1)
	node.InactiveProgram(testNode)
}
