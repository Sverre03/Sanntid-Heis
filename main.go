package main

import (
	"elev/tests"
	"fmt"
	"testing"
)

func main() {
	fmt.Println("Starting test")
	// tests.TestTransmitFunctions()
	t := &testing.T{}
	tests.TestNodeReceivesHallButtonAndProcessesMasterAssignment(t)
}
