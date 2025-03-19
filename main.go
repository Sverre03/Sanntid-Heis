package main

import "elev/node"

func main() {

	mainNode := node.MakeNode(1, "localhost:16456", 20011, 20012)
	secondNode := node.MakeNode(2, "localhost:16457", 20012, 20011)
	mainNode.State = node.Disconnected
	secondNode.State = node.Disconnected
	go func() {
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
	}()
	go func() {
		for {
			switch secondNode.State {
			
			case node.Inactive:
				secondNode.State = node.InactiveProgram(secondNode)

			case node.Disconnected:
				secondNode.State = node.DisconnectedProgram(secondNode)

			case node.Slave:
				secondNode.State = node.SlaveProgram(secondNode)

			case node.Master:
				secondNode.State = node.MasterProgram(secondNode)

			}
		}
	}()
	select {}
}
