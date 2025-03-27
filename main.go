package main

import (
	"elev/node"
	"os"
	"strconv"
)

func main() {

	argsWithoutProg := os.Args[1:]
	elevPort := "localhost:" + argsWithoutProg[0]
	bcastPort, _ := strconv.Atoi(argsWithoutProg[1])
	receiverPort, _ := strconv.Atoi(argsWithoutProg[2])
	id, _ := strconv.Atoi(argsWithoutProg[3])

	mainNode := node.MakeNode(id, elevPort, bcastPort, receiverPort)
	//	mainNode.GlobalHallRequests = [4][2]bool{{true, true}, {true, true}, {true, true}, {true, true}}
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
