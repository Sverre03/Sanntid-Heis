<<<<<<<< HEAD:costFNS/hallRequestAssigner/hallRequestAssigner.go
package hallRequestAssigner
========
package example
>>>>>>>> bcd1e47be0125e70e0fe13f792836e4dfb6d157e:usage-examples/example.go

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

func outputFunction(input HRAInput) *map[string][][2]bool {
	
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
	ret, err := exec.Command("../hallRequestAssigner/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		return nil
	}

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		return nil
	}
	return output
}

