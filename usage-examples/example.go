package example

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
)

// Struct members must be public in order to be accessible by json.Marshal/.Unmarshal
// This means they must start with a capital letter, so we need to use field renaming struct tags to make them camelCase

type HRAElevState struct {
	Behavior    string `json:"behavior"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [][2]bool               `json:"hallRequests"`
	States       map[string]HRAElevState `json:"states"`
}

func inputFunction(states []HRAElevState, HallRequests [][2]bool) HRAInput {
	input := HRAInput{
		HallRequests: HallRequests,
		States:       make(map[string]HRAElevState),
	}
	for i, state := range states {
		input.States[fmt.Sprintf("elevator%d", i)] = state
	}
	return input
}

func main() {

	elevator := HRAElevState{
		Behavior:    "moving",
		Floor:       1,
		Direction:   "down",
		CabRequests: []bool{true, false, false, true},
	}
	elevator1 := HRAElevState{
		Behavior:    "idle",
		Floor:       0,
		Direction:   "stop",
		CabRequests: []bool{false, false, true, false},
	}
	elevator2 := HRAElevState{
		Behavior:    "idle",
		Floor:       0,
		Direction:   "stop",
		CabRequests: []bool{true, false, false, false},
	}

	test := []HRAElevState{elevator, elevator1, elevator2}

	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	jsonBytes, err := json.Marshal(inputFunction(test, [][2]bool{{false, true}, {false, false}, {false, true}, {false, false}}))
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
		return
	}

	ret, err := exec.Command("../hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		return
	}

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		return
	}

	fmt.Printf("output: \n")
	for k, v := range *output {
		fmt.Printf("%6v :  %+v\n", k, v)
	}
}
