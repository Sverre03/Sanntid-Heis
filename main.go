package main

import (
	"elev/tests"
	"fmt"
)

func main() {
	fmt.Println("Starting test")
	//tests.TestHRA()
	// tests.TestTransmitFunctions()
	//t := &testing.T{}
	//tests.TestNodeReceivesHallButtonAndProcessesMasterAssignment(t)
	tests.RunTestNode()
}
