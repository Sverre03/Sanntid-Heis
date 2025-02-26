package testing

import (
	"elev/Network/comm"
	"elev/Network/network/messages"
	"elev/util/config"
	"errors"
	"fmt"
	"time"
)

func transmitDummyData(elevStatesTx chan messages.ElevStates, id int) {
	dummyData := messages.ElevStates{NodeID: id, Direction: "up", Floor: 1, CabRequest: [config.NumFloors]bool{false, false, false, false}, Behavior: "Down"}
	for {
		time.Sleep(50 * time.Millisecond)
		elevStatesTx <- dummyData
	}
}

func TestMasterSlaveACKs() {

	// some channels for communicating with go routines
	err := testAckDistr()
	if err == nil {
		fmt.Println("Ack test passed")
	} else {
		fmt.Println(err.Error())
	}
}

func testAckDistr() error {

	var err error
	ackRx := make(chan messages.Ack)
	hallAssignmentsAck := make(chan messages.Ack)
	lightUpdateAck := make(chan messages.Ack)
	ConnectionReqAck := make(chan messages.Ack)
	CabRequestInfoAck := make(chan messages.Ack)
	HallAssignmentCompleteAck := make(chan messages.Ack)
	timeoutChannel := make(chan int)

	receivedAcks := [5]bool{false, false, false, false, false}

	go comm.IncomingAckDistributor(ackRx, hallAssignmentsAck, lightUpdateAck, ConnectionReqAck, CabRequestInfoAck, HallAssignmentCompleteAck)

	numAckSent := 0
	for i := 0; i < 5; i++ {

		id, e := comm.GenerateMessageID(comm.MessageIDType(i))
		if e != nil {
			err = e
			return err
		}
		ackRx <- messages.Ack{NodeID: i, MessageID: id}
		numAckSent++
	}
	time.AfterFunc(time.Second, func() {
		timeoutChannel <- 1
	})

ForLoop:
	for {
		select {
		case msg := <-hallAssignmentsAck:
			numAckSent--
			if receivedAcks[msg.NodeID] {
				err = errors.New("received two acks on same channel")
				break ForLoop
			}
			receivedAcks[msg.NodeID] = true
		case msg := <-lightUpdateAck:
			numAckSent--
			if receivedAcks[msg.NodeID] {
				err = errors.New("received two acks on same channel")
				break ForLoop
			}
			receivedAcks[msg.NodeID] = true
		case msg := <-ConnectionReqAck:
			numAckSent--
			if receivedAcks[msg.NodeID] {
				err = errors.New("received two acks on same channel")
				break ForLoop
			}
			receivedAcks[msg.NodeID] = true
		case msg := <-CabRequestInfoAck:
			numAckSent--
			if receivedAcks[msg.NodeID] {
				err = errors.New("received two acks on same channel")
				break ForLoop
			}
			receivedAcks[msg.NodeID] = true
		case msg := <-HallAssignmentCompleteAck:
			numAckSent--
			if receivedAcks[msg.NodeID] {
				err = errors.New("received two acks on same channel")
				break ForLoop
			}
			receivedAcks[msg.NodeID] = true
		case <-timeoutChannel:
			if numAckSent > 0 {
				err = fmt.Errorf("not all acks were received, still have %d left", numAckSent)

			}
			break ForLoop
		}
	}

	return err
}
