package main

import (
	"context"
	"fmt"
	"time"
	"test_FSM/master"

	"github.com/looplab/fsm"
)

type Node struct {
	ID  int
	FSM *fsm.FSM

	TOLC                 time.Time
	Elevator             *Elevator
	TaskQueue            []string
	GlobalHallRequests   []string
	LastKnownStates      map[int]string
}

type Elevator struct {
	Floor    int
	Dir      string
	Behavior string
	CabCalls []int
}

func NewNode(id int) *Node {
	n := &Node{
		ID:                 id,
		Elevator:           &Elevator{},
		TaskQueue:          make([]string, 0),
		GlobalHallRequests: make([]string, 0),
		LastKnownStates:    make(map[int]string),
	}

	n.FSM = fsm.NewFSM(
		"inactive",
		fsm.Events{
			{Name: "initialize", Src: []string{"inactive"}, Dst: "disconnected"},
			{Name: "connect", Src: []string{"disconnected"}, Dst: "slave"},
			{Name: "disconnect", Src: []string{"slave", "master"}, Dst: "disconnected"},
			{Name: "promote", Src: []string{"slave", "disconnected"}, Dst: "master"},
			{Name: "demote", Src: []string{"master"}, Dst: "slave"},
			{Name: "inactivate", Src: []string{"slave", "disconnected", "master"}, Dst: "inactive"},
		},

		fsm.Callbacks{
			"enter_state": func(_ context.Context, e *fsm.Event) {
				fmt.Printf("Node %d skiftet fra %s til %s\n", n.ID, e.Src, e.Dst)
			},

			"enter_master": n.onEnterMaster,
		},
	)
	return n
}

func (n *Node) onEnterMaster(_ context.Context, e *fsm.Event) {
	n.TOLC = time.Now()
	fmt.Printf("Node %d er nå MASTER. Med TOLC lik %s \n", n.ID, n.TOLC)
	master.MasterProgram()
}

func main() {
	node := NewNode(1)

	if err := node.FSM.Event(context.Background(), "initialize"); err != nil {
		fmt.Println("Error:", err)
	}
	if err := node.FSM.Event(context.Background(), "promote"); err != nil {
		fmt.Println("Error:", err)
	}
}
