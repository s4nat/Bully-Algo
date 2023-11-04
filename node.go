package main

import (
	"fmt"
	"math/rand"
	"time"
)

const MESSAGE_TIMEOUT = 5
const DRIFT_RATE = 0.05

type Node struct {
	id                 int
	channel            chan Message
	quit               chan bool
	isCoordinator      bool
	localClock         float64
	driftRate          float64
	knownCoordinatorId int
	dead               bool
	electionInvoked    bool
}

type Message struct {
	Type                string
	FromID              int
	Data                float64
	SyncResponseChannel chan bool
	// NackChannel         chan bool
	// NackSent bool
}

func handleIncomingMessage(n *Node, msg *Message, allNodes []*Node) {
	if !n.dead {
		switch msg.Type {
		case "ELECTION":
			handleElection(n, msg, allNodes)
		case "NACK":
			// nothing, just concede election to higher ID node
			handleNack(n, msg)
		case "VICTORY":
			// nothing, just know that sender is now coordinator
			handleVictoryAnnouncement(n, msg)
		case "SYNC REQ":
			handleSyncReq(n, msg, allNodes)
			// periodically sent by any nodes
			// request coordinator to send their data
			// used to detect node failures
		case "SYNC RES":
			handleSyncRes(n, msg)
			// run by node receiving sync data
			// updates its own data with coordinator's data
		}
	}
}

// Send out SYNC REQ messages and wait for SYNC RES messages
func requestSynchronization(n *Node, allNodes []*Node) {
	coordinator := findNodeByID(n.knownCoordinatorId, allNodes)

	// If coordinator left responsibly (removeNode) it will not be in allNodes
	// If coordinator left silently (killNode) it will still be in allNodes
	if coordinator == nil {
		fmt.Printf("ERROR: No known coordinator %d found!", n.knownCoordinatorId)
		return
	}

	// Send synchronization request
	fmt.Printf("\nLOG: Node %d sending Sync Req to Coordinator %d\n", n.id, n.knownCoordinatorId)
	msg := Message{Type: "SYNC REQ", FromID: n.id, SyncResponseChannel: make(chan bool, 10)}
	coordinator.channel <- msg

	timeout := time.After(MESSAGE_TIMEOUT * time.Second)

	select {
	case <-msg.SyncResponseChannel:
		// receive a message, if it's a SYNC RES, it'll be handled in handleIncomingMessage
		fmt.Printf("\nLOG: Node %d received a SYNC RES message\n", n.id)
	case <-timeout:
		fmt.Printf("\nTIMEOUT: Node %d considers Coordinator %d as dead. Initiating election...\n", n.id, coordinator.id)
		initiateElection(n, allNodes)
	}
}

func handleElection(n *Node, msg *Message, allNodes []*Node) {
	// If node is dead, it wont respond to the election
	if n.dead {
		fmt.Printf("\nDEAD NODE: Node %d is dead/unresponsive. Detected in Node %d's election.\n", n.id, msg.FromID)
		return
	}

	// If node is alive, it will handle election
	fmt.Printf("\nLOG: Node %d is handling election request from Node %d...\n", n.id, msg.FromID)

	if n.id > msg.FromID {
		senderNode := findNodeByID(msg.FromID, allNodes)
		if senderNode != nil {
			sendNack(n, msg, allNodes)
			fmt.Printf("\nDEBUG: Node %d has finished sending nack to Node %d\n", n.id, msg.FromID)
			if !n.electionInvoked {
				initiateElection(n, allNodes)
				n.electionInvoked = true
			} else {
				fmt.Printf("\nLOG: Node %d has already initiated an election. Skipping...\n", n.id)
				return
			}
		} else {
			fmt.Printf("\nLOG: Node %d which initiated election has left\n", msg.FromID)
		}
	}
}

// Send out "ELECTION" messages and wait for "NACK" messages
func initiateElection(n *Node, allNodes []*Node) {
	fmt.Printf("\nLOG: Node %d is initiating election...\n", n.id)
	higherNodes := getHigherIDNodes(n.id, allNodes)
	if len(higherNodes) == 0 {
		// If there are no higher nodes, the current node can declare itself as the coordinator.
		becomeCoordinator(n, allNodes)
		return
	}

	// Create an election message with a nack channel attached
	msg := Message{Type: "ELECTION", FromID: n.id}
	// Send election messages to all higher ID nodes
	for _, node := range higherNodes {
		node.channel <- msg
	}

	// Simulate receival of NACK. If higherNodes all dead => no NACK received => can become coordinator
	canBecomeCoordinator := true
	for _, node := range higherNodes {
		if !node.dead {
			canBecomeCoordinator = false
		}
	}
	if canBecomeCoordinator {
		becomeCoordinator(n, allNodes)
	} else {
		fmt.Printf("\nDEFEAT: Node %d will not become the coordinator\n", n.id)
	}
}

func becomeCoordinator(n *Node, allNodes []*Node) {
	fmt.Printf("\nVICTORY: Node %d won the election and is announcing victory...\n", n.id)
	n.isCoordinator = true
	n.knownCoordinatorId = n.id
	for _, node := range allNodes {
		if node.id != n.id {
			node.channel <- Message{Type: "VICTORY", FromID: n.id}
		}
	}
	n.electionInvoked = false
}

func handleNack(n *Node, msg *Message) {
	fmt.Printf("\nNACK: Node %d received NACK from Node %d\n", n.id, msg.FromID)
}

func handleVictoryAnnouncement(n *Node, msg *Message) {
	n.knownCoordinatorId = msg.FromID
	n.electionInvoked = false
	fmt.Printf("\nACCEPTANCE: Node %d accepts Node %d as coordinator\n", n.id, msg.FromID)
}

// Sending "SYNC RES" to requestingNode.Channel
// Signalling that "SYNC RES" has been sent to msg.NackChannel
func handleSyncReq(n *Node, msg *Message, allNodes []*Node) {
	if n.isCoordinator && !n.dead {
		fmt.Printf("\nCoordinator Node %d is sending its local clock\n", n.id)
		requestingNode := findNodeByID(msg.FromID, allNodes)
		requestingNode.channel <- Message{Type: "SYNC RES", FromID: n.id, Data: n.localClock}
		// Signal that SYNC RES has been sent
		msg.SyncResponseChannel <- true
	}
}

func handleSyncRes(n *Node, msg *Message) {
	if msg.FromID == n.knownCoordinatorId {
		fmt.Printf("\nNode %d has updated its local clock to %f from Coordinator %d\n", n.id, msg.Data, msg.FromID)
		n.localClock = msg.Data
	}
}

func sendNack(n *Node, msg *Message, allNodes []*Node) {
	// Signal that NACK has been sent
	// fmt.Println("DEBUG 198:", msg, "in Node", n.id)
	// msg.NackChannel <- true
	// msg.NackSent = true
	fmt.Printf("\nLOG: Node %d sending NACK to Node %d\n", n.id, msg.FromID)
	targetNode := findNodeByID(msg.FromID, allNodes)
	targetNode.channel <- Message{Type: "NACK", FromID: n.id}
}

func findNodeByID(id int, allNodes []*Node) *Node {
	for _, node := range allNodes {
		if node.id == id {
			return node
		}
	}
	return nil
}

func getHigherIDNodes(currentID int, allNodes []*Node) []*Node {
	var higherNodes []*Node
	for _, node := range allNodes {
		if node.id > currentID {
			higherNodes = append(higherNodes, node)
		}
	}
	return higherNodes
}

func runClock(n *Node) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			drift := (rand.Float64()*2 - 1) * n.driftRate
			n.localClock += 1 + drift
		}
	}

}

func runNode(nodeCounter *int) *Node {
	// Initialization
	nodeChannel := make(chan Message, 100)
	quit := make(chan bool)
	node := &Node{id: *nodeCounter,
		channel:            nodeChannel,
		quit:               quit,
		isCoordinator:      false,
		localClock:         0,
		driftRate:          DRIFT_RATE,
		knownCoordinatorId: -1}

	// Increment node counter
	*nodeCounter++

	// Start necessary go routines
	go runClock(node)
	go func() {
		for {
			select {
			case msg := <-node.channel:
				handleIncomingMessage(node, &msg, allNodes)
			case <-node.quit:
				fmt.Printf("Node %d shutting down...\n", node.id)
				close(node.channel)
				return
			}
		}
	}()

	// return the node
	return node
}
