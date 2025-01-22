package messages

const N_FLOORS int = 4

// a struct for acknowledging a message as received
type AckMsg struct {
	MessageId  int
	ReceiverId string
}

type AssignmentMsg struct {
	HallButtons [N_FLOORS][2]bool
	MessageId   int
}
