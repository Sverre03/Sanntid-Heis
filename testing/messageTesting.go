package test

import (
	transmitters "elev/Network/elev_communication"
	"elev/Network/network/bcast"
	"elev/Network/network/messages"
	"elev/util/config"
)

const slavePort = 20011
const masterPort = 20012

func TestMasterSlaveACKs() {

	go dummyMaster(masterPort, slavePort, 1)
	go dummySlave(slavePort, masterPort, 2)
	// wait for both to finish
}

func dummyMaster(sendPort, receivePort, id int) bool {
	error := false
	ackTx := make(chan messages.Ack)
	ackRx := make(chan messages.Ack)
	hallAssignmentsAck := make(chan messages.Ack)
	lightUpdateAck := make(chan messages.Ack)
	ConnectionReqAck := make(chan messages.Ack)
	CabRequestInfoAck := make(chan messages.Ack)
	HallAssignmentCompleteAck := make(chan messages.Ack)

	ElevStatesTx := make(chan messages.ElevStates)
	ElevStatesRx := make(chan messages.ElevStates)

	HallAssignmentsTx := make(chan messages.NewHallAssignments)
	HallAssignmentsRx := make(chan messages.NewHallAssignments)
	outgoingHallAssignments := make(chan messages.NewHallAssignments)

	CabRequestInfoTx := make(chan messages.CabRequestINF)
	CabRequestInfoRx := make(chan messages.CabRequestINF)

	GlobalHallRequestTx := make(chan messages.GlobalHallRequest)
	GlobalHallRequestRx := make(chan messages.GlobalHallRequest)

	HallLightUpdateTx := make(chan messages.HallLightUpdate)
	HallLightUpdateRx := make(chan messages.HallLightUpdate)
	// outgoingLightUpdate := make(chan messages.HallLightUpdate)

	ConnectionReqTx := make(chan messages.ConnectionReq)
	ConnectionReqRx := make(chan messages.ConnectionReq)

	HallAssignmentCompleteTx := make(chan messages.HallAssignmentComplete)
	HallAssignmentCompleteRx := make(chan messages.HallAssignmentComplete)

	go bcast.Transmitter(sendPort, ackTx, ElevStatesTx, HallAssignmentsTx, CabRequestInfoTx, GlobalHallRequestTx, HallLightUpdateTx, ConnectionReqTx, HallAssignmentCompleteTx)
	go bcast.Receiver(receivePort, ackRx, ElevStatesRx, HallAssignmentsRx, CabRequestInfoRx, GlobalHallRequestRx, HallLightUpdateRx, ConnectionReqRx, HallAssignmentCompleteRx)

	go transmitters.IncomingAckDistributor(ackRx, hallAssignmentsAck, lightUpdateAck, ConnectionReqAck, CabRequestInfoAck, HallAssignmentCompleteAck)

	go transmitters.HallAssignmentsTransmitter(HallAssignmentsTx, outgoingHallAssignments, hallAssignmentsAck)
	// go transmitters.LightUpdateTransmitter(HallLightUpdateTx, outgoingLightUpdate, lightUpdateAck)

	outgoingHallAssignments <- messages.NewHallAssignments{NodeID: id, HallAssignment: [config.NumFloors][2]bool{{false, false}, {false, false}, {false, false}, {false, false}}, MessageID: 0}

	return error
}

func dummySlave(sendPort, receivePort, id int) bool {
	error := false
	AckTx := make(chan messages.Ack)
	AckRx := make(chan messages.Ack)

	ElevStatesTx := make(chan messages.ElevStates)
	ElevStatesRx := make(chan messages.ElevStates)

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

	go bcast.Transmitter(sendPort, AckTx, ElevStatesTx, HallAssignmentsTx, CabRequestInfoTx, GlobalHallRequestTx, HallLightUpdateTx, ConnectionReqTx, HallAssignmentCompleteTx)
	go bcast.Receiver(receivePort, AckRx, ElevStatesRx, HallAssignmentsRx, CabRequestInfoRx, GlobalHallRequestRx, HallLightUpdateRx, ConnectionReqRx, HallAssignmentCompleteRx)

	for {

	}
	return error
}
