package main

import (
	"elev/node"
	"os"
	"strconv"
)

func main() {
	
	// Initialize node with a elevator port, broadcast port for transmitting and receiving, unique id for node
	argsWithoutProg := os.Args[1:]
	elevPort := "localhost:" + argsWithoutProg[0]
	bcastPort, _ := strconv.Atoi(argsWithoutProg[1])
	id, _ := strconv.Atoi(argsWithoutProg[2])

	mainNode := node.MakeNode(id, elevPort, bcastPort)
	mainNode.State = node.Inactive

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
