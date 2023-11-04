package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var allNodes = []*Node{}
var nodeCounter = 0

func startNodeCLI() {
	node := runNode(&nodeCounter)
	allNodes = append(allNodes, node)
	fmt.Printf("Node %d started...\n", node.id)
}

func removeNodeCLI(id int) {
	// Locate and responsibly "remove" the node. Remove node from allNodes list
	for index, node := range allNodes {
		if node.id == id {
			allNodes = append(allNodes[:index], allNodes[index+1:]...)
			node.quit <- true
			fmt.Printf("Node %d removed\n", id)
			return
		}
	}
	fmt.Printf("Node %d not found\n", id)
}

func killNodeCLI(id int) {
	for _, node := range allNodes {
		node.electionInvoked = false
	}
	node := findNodeByID(id, allNodes)
	node.dead = true
	fmt.Printf("\nNode %d has now been killed\n", id)
}

func requestSyncCLI(id int) {
	for _, node := range allNodes {
		node.electionInvoked = false
	}
	for _, node := range allNodes {
		if node.id == id {
			requestSynchronization(node, allNodes)
			return
		}
	}
	fmt.Printf("Node %d not found\n", id)
}

func initiateElectionCLI(id int) {
	// for _, node := range allNodes {
	// 	node.electionInvoked = false
	// }
	for _, node := range allNodes {
		if node.id == id {
			initiateElection(node, allNodes)
			return
		}
	}
	fmt.Printf("Node %d not found\n", id)
}

func listNodesCLI() {
	for _, node := range allNodes {
		fmt.Println("NodeID:", node.id)
		fmt.Println("Local Clock:", node.localClock)
		fmt.Println("Known Coordinator:", node.knownCoordinatorId)
		fmt.Println("Dead:", node.dead)
		fmt.Println("Election Invoked:", node.electionInvoked)
		fmt.Println("\n")
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		// Print a prompt
		fmt.Print("> ")

		// read input from user
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
		}

		// Parse the input to handle commands
		handleInput(strings.TrimSpace(input))
	}
}

func handleInput(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	command := parts[0]
	parameters := parts[1:]

	switch command {
	case "startNode":
		startNodeCLI()

	case "removeNode":
		if len(parameters) != 1 {
			fmt.Println("Usage: removeNode <nodeID>")
			return
		}
		nodeID, err := strconv.Atoi(parameters[0])
		if err != nil {
			fmt.Println("Error: nodeID should be an integer")
		}
		removeNodeCLI(nodeID)

	case "killNode":
		if len(parameters) != 1 {
			fmt.Println("Usage: removeNode <nodeID>")
			return
		}
		nodeID, err := strconv.Atoi(parameters[0])
		if err != nil {
			fmt.Println("Error: nodeID should be an integer")
		}
		killNodeCLI(nodeID)

	case "initiateElection":
		if len(parameters) != 1 {
			fmt.Println("Usage: initiateElection <nodeID>")
			return
		}
		nodeID, err := strconv.Atoi(parameters[0])
		if err != nil {
			fmt.Println("Error: nodeID should be an integer")
			return
		}
		initiateElectionCLI(nodeID)

	case "requestSync":
		if len(parameters) != 1 {
			fmt.Println("Usage: requestSync <nodeID>")
			return
		}
		nodeID, err := strconv.Atoi(parameters[0])
		if err != nil {
			fmt.Println("Error: nodeID should be an integer")
			return
		}
		requestSyncCLI(nodeID)
	case "listNodes":
		listNodesCLI()

	case "multiElect":
		startNodeCLI()
		startNodeCLI()
		startNodeCLI()
		startNodeCLI()
		startNodeCLI()

		go initiateElectionCLI(0)
		go initiateElectionCLI(1)
		listNodesCLI()

	case "failNonCoordinatorDuringElection":
		startNodeCLI()
		startNodeCLI()
		startNodeCLI()
		startNodeCLI()
		startNodeCLI()

		go initiateElectionCLI(0)
		go killNodeCLI(2) // non elector node
		listNodesCLI()

	case "silentLeave":
		startNodeCLI()
		startNodeCLI()
		startNodeCLI()
		startNodeCLI()
		startNodeCLI()

		killNodeCLI(3)
		listNodesCLI()

	default:
		fmt.Println("Unknown command:", command)
	}

}
