package testing

import (
	"elev/Network/comm"
	"elev/Network/network/bcast"
	"elev/Network/network/messages"
	"elev/util/config"
	"errors"
	"fmt"
	"time"
)

const slavePort = 20011
const masterPort = 20012

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
			break ForLoop
		}
	}

	if numAckSent > 0 {
		err = errors.New(fmt.Sprintf("not all acks were received, still have %d left", numAckSent))
	}

	return err
}

func dummyMaster(sendPort, receivePort, id int, killCh chan bool, errorCh chan error) {
	var err error
	fmt.Println(id)

	ackRx := make(chan messages.Ack)
	hallAssignmentsAck := make(chan messages.Ack)
	lightUpdateAck := make(chan messages.Ack)
	ConnectionReqAck := make(chan messages.Ack)
	CabRequestInfoAck := make(chan messages.Ack)
	HallAssignmentCompleteAck := make(chan messages.Ack)

	// elevStatesTx := make(chan messages.ElevStates)
	elevStatesRx := make(chan messages.ElevStates)

	HallAssignmentsTx := make(chan messages.NewHallAssignments)
	outgoingHallAssignments := make(chan messages.NewHallAssignments)

	HallLightUpdateTx := make(chan messages.HallLightUpdate)
	outgoingLightUpdate := make(chan messages.HallLightUpdate)

	commandCh := make(chan string, 10)
	timeOfLastContactCh := make(chan time.Time)
	elevStatesCh := make(chan map[int]messages.ElevStates)
	activeNodeIDsCh := make(chan []int)

	go comm.IncomingAckDistributor(ackRx, hallAssignmentsAck, lightUpdateAck, ConnectionReqAck, CabRequestInfoAck, HallAssignmentCompleteAck)

	go comm.HallAssignmentsTransmitter(HallAssignmentsTx, outgoingHallAssignments, hallAssignmentsAck)

	go comm.ElevStatesListener(commandCh,
		timeOfLastContactCh,
		elevStatesCh,
		activeNodeIDsCh,
		elevStatesRx)

	// go transmitDummyData(elevStatesTx, id)

	go comm.LightUpdateTransmitter(HallLightUpdateTx,
		outgoingLightUpdate,
		lightUpdateAck,
		commandCh,
		activeNodeIDsCh)

	outgoingHallAssignments <- messages.NewHallAssignments{NodeID: 1, HallAssignment: [config.NumFloors][2]bool{{false, false}, {false, false}, {false, false}, {false, false}}, MessageID: 0}
	// send smth to slave
	// time.Sleep(time.Millisecond)
	// outgoingHallAssignments <- messages.NewHallAssignments{NodeID: 1, HallAssignment: [config.NumFloors][2]bool{{false, false}, {false, false}, {false, false}, {false, false}}, MessageID: 0}

	// listen on this killChannel
	<-killCh

	errorCh <- err
}

func dummySlave(sendPort, receivePort, id int, killCh chan bool, errorCh chan error) {
	var err error
	fmt.Println(id)
	ackTx := make(chan messages.Ack)
	ackRx := make(chan messages.Ack)

	hallAssignmentsAck := make(chan messages.Ack)
	lightUpdateAck := make(chan messages.Ack)
	ConnectionReqAck := make(chan messages.Ack)
	CabRequestInfoAck := make(chan messages.Ack)
	HallAssignmentCompleteAck := make(chan messages.Ack)

	elevStatesTx := make(chan messages.ElevStates)
	elevStatesRx := make(chan messages.ElevStates)

	HallAssignmentsTx := make(chan messages.NewHallAssignments)
	HallAssignmentsRx := make(chan messages.NewHallAssignments)

	CabRequestInfoTx := make(chan messages.CabRequestINF)
	CabRequestInfoRx := make(chan messages.CabRequestINF)

	GlobalHallRequestTx := make(chan messages.GlobalHallRequest)
	GlobalHallRequestRx := make(chan messages.GlobalHallRequest)

	HallLightUpdateTx := make(chan messages.HallLightUpdate)
	HallLightUpdateRx := make(chan messages.HallLightUpdate)

	ConnectionReqTx := make(chan messages.ConnectionReq)
	ConnectionReqRx := make(chan messages.ConnectionReq)

	HallAssignmentCompleteTx := make(chan messages.HallAssignmentComplete)
	HallAssignmentCompleteRx := make(chan messages.HallAssignmentComplete)

	commandCh := make(chan string, 10)
	timeOfLastContactCh := make(chan time.Time)
	elevStatesCh := make(chan map[int]messages.ElevStates)
	activeNodeIDsCh := make(chan []int)

	go bcast.Transmitter(sendPort, ackTx, elevStatesTx, HallAssignmentsTx, CabRequestInfoTx, GlobalHallRequestTx, HallLightUpdateTx, ConnectionReqTx, HallAssignmentCompleteTx)
	go bcast.Receiver(receivePort, ackRx, elevStatesRx, HallAssignmentsRx, CabRequestInfoRx, GlobalHallRequestRx, HallLightUpdateRx, ConnectionReqRx, HallAssignmentCompleteRx)

	go comm.IncomingAckDistributor(ackRx, hallAssignmentsAck, lightUpdateAck, ConnectionReqAck, CabRequestInfoAck, HallAssignmentCompleteAck)

	go comm.ElevStatesListener(commandCh, timeOfLastContactCh, elevStatesCh, activeNodeIDsCh, elevStatesRx)

	go transmitDummyData(elevStatesTx, id)

	// now, we must listen.
	hAsCounter := 0
	var lastReceivedHAs messages.NewHallAssignments
	err = errors.New("message receival failed")
OuterLoop:
	for {
		select {
		case newHAs := <-HallAssignmentsRx:
			hAsCounter += 1
			if newHAs == lastReceivedHAs {
				hAsCounter += 1
			}

			if hAsCounter == 10 {
				ackRx <- messages.Ack{MessageID: newHAs.MessageID, NodeID: newHAs.NodeID}
				err = nil
			}

			lastReceivedHAs = newHAs

		case <-killCh:
			errorCh <- err
			break OuterLoop
		}
	}

}
