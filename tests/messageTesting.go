package tests

import (
	"elev/Network/messagehandler"
	"elev/Network/messages"
	"elev/util/config"
	"errors"
	"fmt"
	"time"
)

func testMessageIDGenerator() error {
	for i := uint64(0); i < 4; i++ {
		if j, _ := messagehandler.GenerateMessageID(messagehandler.MessageIDType(i)); j > (i+1)*config.MSG_ID_PARTITION_SIZE || j < i*config.MSG_ID_PARTITION_SIZE {
			return fmt.Errorf("message id outside value area for messagetype %d", i)
		}

	}
	return nil
}

func TestTransmitFunctions() {
	var err error

	err = testGlobalHallReqTransmitter()
	if err == nil {
		fmt.Println("hall assignment transmitter test passed")
	} else {
		fmt.Println(err.Error())
		return
	}

	err = testHAss()
	if err == nil {
		fmt.Println("Hall assignment test passed")
	} else {
		fmt.Println(err.Error())
		return
	}

	err = testMessageIDGenerator()
	if err == nil {
		fmt.Println("MessageID generator test passed")
	} else {
		fmt.Println(err.Error())
		return
	}

	err = testAckDistr()
	if err == nil {
		fmt.Println("Ack distributor test passed")
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

	go messagehandler.HallAssignmentsTransmitter(HallAssignmentsTx, OutgoingNewHallAssignments, HallAssignmentsAck)

	dummyHallAssignment1 := messages.NewHallAssignments{NodeID: id, HallAssignment: [config.NUM_FLOORS][2]bool{{false, false}, {false, false}, {false, false}, {false, false}}, MessageID: 0}
	dummyHallAssignment2 := messages.NewHallAssignments{NodeID: id + 1, HallAssignment: [config.NUM_FLOORS][2]bool{{false, false}, {false, false}, {false, false}, {false, false}}, MessageID: 0}

	OutgoingNewHallAssignments <- dummyHallAssignment1
	OutgoingNewHallAssignments <- dummyHallAssignment2

	numMsgReceived := 0
	hasReceived := false

	time.AfterFunc(5*time.Second, func() {
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

func testGlobalHallReqTransmitter() error {
	var err error
	transmitEnableCh := make(chan bool, 1)
	GlobalHallRequestTx := make(chan messages.GlobalHallRequest, 1)
	requestsForBroadcastCh := make(chan messages.GlobalHallRequest, 1)
	timeoutChannel := make(chan int, 1)

	haveReceived := false

	go messagehandler.GlobalHallRequestsTransmitter(transmitEnableCh, GlobalHallRequestTx, requestsForBroadcastCh)

	var currentHallRequests [config.NUM_FLOORS][2]bool

	time.AfterFunc(5*time.Second, func() {
		timeoutChannel <- 10
	})

	time.AfterFunc(2*time.Second, func() {
		timeoutChannel <- 5
	})

	time.AfterFunc(150*time.Millisecond, func() {
		timeoutChannel <- 1
	})

	requestsForBroadcastCh <- messages.GlobalHallRequest{HallRequests: currentHallRequests}
	transmitEnableCh <- true

ForLoop:
	for {
		select {
		case GHallReq := <-GlobalHallRequestTx:

			if currentHallRequests != GHallReq.HallRequests {
				err = errors.New("received wrong hall requests")
				fmt.Println(GHallReq.HallRequests)
				break ForLoop
			}
			haveReceived = true

		case i := <-timeoutChannel:

			if !haveReceived {
				err = errors.New("did not receive an update in time")
				break ForLoop
			}
			if i == 10 {
				break ForLoop
			} else if i == 5 {
				fmt.Println("Updating new hall requests")
				currentHallRequests[0][1] = true
				requestsForBroadcastCh <- messages.GlobalHallRequest{HallRequests: currentHallRequests}
			}

			time.AfterFunc(config.MASTER_TIMEOUT, func() {
				timeoutChannel <- 1
			})
			haveReceived = false
		}
	}
	return err
}

func testAckDistr() error {

	var err error
	// if these channels are not buffered, the listener is blocking while waiting to send the first message (waiting for someone to listen) and so we get a deadlock.
	ackRx := make(chan messages.Ack, 1)
	hallAssignmentsAck := make(chan messages.Ack, 1)
	lightUpdateAck := make(chan messages.Ack, 1)
	ConnectionReqAck := make(chan messages.Ack, 1)
	HallAssignmentCompleteAck := make(chan messages.Ack, 1)
	timeoutChannel := make(chan int)

	receivedAcks := [4]bool{false, false, false, false}

	go messagehandler.IncomingAckDistributor(ackRx, hallAssignmentsAck, lightUpdateAck, ConnectionReqAck, HallAssignmentCompleteAck)

	numAckSent := 0
	for i := 0; i < 4; i++ {

		id, e := messagehandler.GenerateMessageID(messagehandler.MessageIDType(i))

		if e != nil {
			err = e
			return err
		}
		// this blocked until I gave all channels an explicit buffer of 1. See reason above
		ackRx <- messages.Ack{NodeID: i, MessageID: id}
		numAckSent++
	}
	time.AfterFunc(1*time.Second, func() {
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
