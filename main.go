package main

import "elev/node"

func main() {

	mainNode := node.MakeNode(1)
	mainNode.State = node.Master
	for {

		switch mainNode.State {

		case node.Inactive:
			mainNode.State = node.InactiveProgram(mainNode)

		case node.Disconnected:
			mainNode.State = node.DisconnectedProgram(mainNode)

		case node.Slave:
			mainNode.State = node.SlaveProgram(mainNode)

		case node.Master:
			mainNode.State = node.MasterProgram(mainNode)

		}
	}
}
