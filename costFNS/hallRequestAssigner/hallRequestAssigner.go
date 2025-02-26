package hallRequestAssigner

import (
	"elev/Network/network/messages"
	"elev/elevator"
	"elev/util/config"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Struct members must be public in order to be accessible by json.Marshal/.Unmarshal
// This means they must start with a capital letter, so we need to use field renaming struct tags to make them camelCase

type HRAElevState struct {
	Behavior    string                  `json:"behavior"`
	Floor       int                     `json:"floor"`
	Direction   string                  `json:"direction"`
	CabRequests [config.NUM_FLOORS]bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [config.NUM_FLOORS][2]bool `json:"hallRequests"`
	States       map[string]HRAElevState    `json:"states"`
}

func InputFunction(allElevStates map[int]messages.ElevStates, hallRequests [config.NUM_FLOORS][2]bool) HRAInput {
	allElevStatesInputFormat := make(map[string]HRAElevState)
	for id, state := range allElevStates {
		allElevStatesInputFormat[fmt.Sprintf("%d", id)] = HRAElevState{
			Behavior:    state.Behavior,
			Floor:       state.Floor,
			Direction:   strings.ToLower(elevator.MotorDirectionToString(state.Direction)),
			CabRequests: state.CabRequest,
		}
	}
	input := HRAInput{
		HallRequests: hallRequests,
		States:       allElevStatesInputFormat,
	}
	return input
}

func OutputFunction(input HRAInput) *map[string][config.NUM_FLOORS][2]bool {

	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
		return nil
	}
	ret, err := exec.Command("costFNS/hallRequestAssigner/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		return nil
	}

	output := new(map[string][config.NUM_FLOORS][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		return nil
	}
	return output
}
