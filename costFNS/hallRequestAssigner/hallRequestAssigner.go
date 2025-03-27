package hallRequestAssigner

import (
	"elev/config"
	"elev/elevator"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// States formatted for the Hall Request Assigner
type HRAElevState struct {
	Behavior    string                  `json:"behavior"`
	Floor       int                     `json:"floor"`
	Direction   string                  `json:"direction"`
	CabRequests [config.NUM_FLOORS]bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool `json:"hallRequests"` // All system hall requests
	States       map[string]HRAElevState                          `json:"states"`
}

func HRAalgorithm(allElevStates map[int]elevator.ElevatorStateReport, hallRequests [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool) map[int][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool {
	allElevStatesInputFormat := make(map[string]HRAElevState)

	for id, nodeState := range allElevStates {
		allElevStatesInputFormat[fmt.Sprintf("%d", id)] = HRAElevState{
			Behavior:    nodeState.Behavior.String(),
			Floor:       nodeState.Floor,
			Direction:   strings.ToLower(nodeState.Direction.String()),
			CabRequests: nodeState.CabRequests,
		}
	}
	input := HRAInput{
		HallRequests: hallRequests,
		States:       allElevStatesInputFormat,
	}
	// fmt.Printf("HRAalgorithm input: %v\n", input)

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
	// fmt.Printf("jsonBytes: %v\n", string(jsonBytes))
	ret, err := exec.Command("costFNS/hallRequestAssigner/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		return nil
	}

	HRAoutput := new(map[string][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool)
	err = json.Unmarshal(ret, &HRAoutput)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		return nil
	}

	HRAoutputFormatting := make(map[int][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool) // Convert map[string][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool to map[int][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool
	for id, output := range *HRAoutput {
		id, err := strconv.Atoi(id)
		if err != nil {
			fmt.Println("Error: ", err)
		}
		HRAoutputFormatting[id] = output
	}
	return HRAoutputFormatting // map[int][config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool
}
