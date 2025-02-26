package tests

import (
	"elev/Network/network/messages"
	"elev/costFNS/hallRequestAssigner"
	"elev/elevator"
	"elev/util/config"
	"fmt"
)

func TestHRA() {
	var newMessage1 messages.ElevStates
	var newMessage2 messages.ElevStates
	var GlobalHallRequest [config.NUM_FLOORS][2]bool

	for i := 0; i < config.NUM_FLOORS; i++ {
		for j := 0; j < 2; j++ {
			GlobalHallRequest[i][j] = false
		}
	}

	GlobalHallRequest[1][0] = true
	GlobalHallRequest[2][1] = true
	GlobalHallRequest[3][0] = true

	fmt.Printf("GlobalHallRequest: %v\n", GlobalHallRequest)

	newMessage1 = messages.ElevStates{NodeID: 1, Direction: elevator.MD_Up, Floor: 1, CabRequest: [config.NUM_FLOORS]bool{false, false, false, false}, Behavior: "idle"}
	newMessage2 = messages.ElevStates{NodeID: 2, Direction: elevator.MD_Down, Floor: 2, CabRequest: [config.NUM_FLOORS]bool{false, false, false, false}, Behavior: "idle"}

	allElevStates := make(map[int]messages.ElevStates)
	allElevStates[0] = newMessage1
	allElevStates[1] = newMessage2

	input := hallRequestAssigner.InputFunction(allElevStates, GlobalHallRequest)
	output := hallRequestAssigner.OutputFunction(input)
	fmt.Printf("Output: %v\n", output)
}
