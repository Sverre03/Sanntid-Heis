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
	dummyData := messages.ElevStates{NodeID: id, Direction: "up", Floor: 1, CabRequest: [config.NUM_FLOORS]bool{false, false, false, false}, Behavior: "Down"}
	for {
		time.Sleep(50 * time.Millisecond)
		elevStatesTx <- dummyData
	}
}

func crazy() {
	for {
		fmt.Println("Test is active")
		time.Sleep(time.Millisecond * 500)
	}
}

func testMessageIDGenerator() error {
	for i := 0; i < 5; i++ {
		if j, _ := comm.GenerateMessageID(comm.MessageIDType(i)); j > (i+1)*config.MSG_ID_PARTITION_SIZE || j < i*config.MSG_ID_PARTITION_SIZE {
			return fmt.Errorf("message id outlide value area for messagetype %d", i)
		}

	}
	return nil
}

func TestMasterSlaveACKs() {
	var err error
	err = testMessageIDGenerator()
	if err == nil {
		fmt.Println("Message id generator test passed")
	} else {
		fmt.Println(err.Error())
		return
	}

	err = testAckDistr()
	if err == nil {
		fmt.Println("Ack test passed")
	} else {
		fmt.Println(err.Error())
		return
	}
}

func testAckDistr() error {

	var err error
	ackRx := make(chan messages.Ack, 1)
	hallAssignmentsAck := make(chan messages.Ack, 1)
	lightUpdateAck := make(chan messages.Ack, 1)
	ConnectionReqAck := make(chan messages.Ack, 1)
	CabRequestInfoAck := make(chan messages.Ack, 1)
	HallAssignmentCompleteAck := make(chan messages.Ack, 1)
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
		// this blocked until I gave all channels an explicit buffer of 1
		ackRx <- messages.Ack{NodeID: i, MessageID: id}
		numAckSent++
	}
	time.AfterFunc(time.Second*1, func() {
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
				fmt.Println(receivedAcks)

			}
			break ForLoop
		}
	}

	return err
}
