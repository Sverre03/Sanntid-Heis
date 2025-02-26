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
	// dummyData := messages.ElevStates{NodeID: id, Direction: "up", Floor: 1, CabRequest: [config.NUM_FLOORS]bool{false, false, false, false}, Behavior: "Down"}
	for {
		time.Sleep(50 * time.Millisecond)
		// elevStatesTx <- dummyData
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

func TestTransmitFunctions() {
	var err error

	err = testHAss()
	if err == nil {
		fmt.Println("Hall assignment test passed")
	} else {
		fmt.Println(err.Error())
		return
	}

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

func testHAss() error {
	id := 10
	err := errors.New("no messages were received")
	timeoutChannel := make(chan int, 1)
	HallAssignmentsTx := make(chan messages.NewHallAssignments, 1)
	OutgoingNewHallAssignments := make(chan messages.NewHallAssignments, 1)
	HallAssignmentsAck := make(chan messages.Ack, 1)

	go comm.HallAssignmentsTransmitter(HallAssignmentsTx, OutgoingNewHallAssignments, HallAssignmentsAck)

	dummyHallAssignment1 := messages.NewHallAssignments{NodeID: id, HallAssignment: [config.NUM_FLOORS][2]bool{{false, false}, {false, false}, {false, false}, {false, false}}, MessageID: 0}
	dummyHallAssignment2 := messages.NewHallAssignments{NodeID: id + 1, HallAssignment: [config.NUM_FLOORS][2]bool{{false, false}, {false, false}, {false, false}, {false, false}}, MessageID: 0}

	OutgoingNewHallAssignments <- dummyHallAssignment1
	OutgoingNewHallAssignments <- dummyHallAssignment2

	numMsgReceived := 0
	hasReceived := false

	time.AfterFunc(time.Second*5, func() {
		timeoutChannel <- 1
	})

ForLoop:
	for {
		select {
		case HAss := <-HallAssignmentsTx:
			switch HAss.NodeID {
			case id + 1:
				if hasReceived {
					err = errors.New("received a message twice that should have been acked")
					break ForLoop
				}
				HallAssignmentsAck <- messages.Ack{NodeID: (id + 1), MessageID: HAss.MessageID}
				hasReceived = true

			case id:

				err = fmt.Errorf("only received %d messages", numMsgReceived)
				numMsgReceived++

				if numMsgReceived > 6 {
					err = fmt.Errorf("keeps resending after messages was supposed to be acked")
					break ForLoop
				}

				if numMsgReceived == 5 {
					HallAssignmentsAck <- messages.Ack{NodeID: id, MessageID: HAss.MessageID}
					err = nil
				}
			}
		case <-timeoutChannel:
			break ForLoop
		}
	}
	return err

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
