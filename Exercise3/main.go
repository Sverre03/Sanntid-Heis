package main

import (
	"Network/network/bcast"
	"Network/network/messages"
	"Network/network/peers"
)

func sendAssignmentMessage(assignmentTx chan messages.AssignmentMsg, ackRx chan messages.AckMsg, msg *messages.AssignmentMsg, peers []string) {
	assignmentTx <- *msg
	// here, you have to wait for ack
	// you will also have to have a timeout
}

// A Node takes an id (any string) and a port.
// In case of assignment received from network, sends it to channel incomingAssignments
// In case of assignment received on outgoingAssignments channel, broadcasts it to the network
func Node(id string, port int, outgoingAssignments <-chan messages.AssignmentMsg, incomingAssignments chan<- messages.AssignmentMsg) {
	peerTxEnable := make(chan bool)

	peerUpdateCh := make(chan peers.PeerUpdate)
	assignmentRx := make(chan messages.AssignmentMsg)
	assignmentTx := make(chan messages.AssignmentMsg)
	ackTx := make(chan messages.AckMsg)
	ackRx := make(chan messages.AckMsg)
	// transmit the fact that you are alive
	go peers.Transmitter(port, id, peerTxEnable)

	// receive updates on other living elevators
	go peers.Receiver(port, peerUpdateCh)

	//
	go bcast.Transmitter(port, assignmentTx, ackTx)
	go bcast.Receiver(port, assignmentRx, ackTx)

	var msg messages.AssignmentMsg
	var peers peers.PeerUpdate

	select {

	case p := <-peerUpdateCh:
		peers = p

	case msg = <-outgoingAssignments:
		// here, you make a subroutine for sending a message. This subroutine will live until the message is accepted from all
		go sendAssignmentMessage(assignmentTx, ackRx, &msg, peers.Peers)
	case msg = <-assignmentRx:
		// ackknowledge the message
		ackTx <- messages.AckMsg{msg.MessageId, id}
		incomingAssignments <- msg
	}
}

func main() {
	msgTx1 := make(chan messages.AssignmentMsg)
	msgRx1 := make(chan messages.AssignmentMsg)
	msgRx2 := make(chan messages.AssignmentMsg)
	msgTx2 := make(chan messages.AssignmentMsg)
	go Node("id1", 20011, msgTx1, msgRx1)
	go Node("id2", 20011, msgTx2, msgRx2)

	select {
	// her kan vi nå sende og receive messages. Prøv å sende mellom to noder og se hva som skjer
	}
}
