package node

import (
	"context"
	"fmt"
)

func InactiveProgram(node *NodeData) {
	fmt.Printf("Node %d is now Inactive\n", node.ID)
	if err := node.NodeState.Event(context.Background(), "initialize"); err != nil {
		fmt.Println("Error:", err)
	}
}
