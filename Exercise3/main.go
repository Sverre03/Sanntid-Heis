package main

import (
	"fmt"
	"os/exec"
)

func main() {
	// Create a new command
	cmd := exec.Command("ls", "-l")

	// Run the command
	output, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print the output
	fmt.Println(string(output))
}