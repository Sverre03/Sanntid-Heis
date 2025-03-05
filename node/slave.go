package node

import "fmt"

func SlaveProgram(node *NodeData) {
	fmt.Printf("Node %d is now a Slave\n", node.ID)

	for {
		select {
		// case
		}

	}
}
