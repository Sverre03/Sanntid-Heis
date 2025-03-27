package config

import (
	"time"
)

// Physical elevator parameters
const NUM_FLOORS = 4
const NUM_BUTTONS = 3
const NUM_HALL_BUTTONS = 2

const HARDWARE_POLL_INTERVAL = 20 * time.Millisecond

const MSG_ID_PARTITION_SIZE = uint64(2 << 60)

// Timing constants for the elevator program
const DOOR_OPEN_DURATION = 3 * time.Second
const DOOR_STUCK_DURATION = 7 * time.Second
const ELEV_STUCK_TIMEOUT = 4 * time.Second
const ELEV_STUCK_POLL_INTERVAL = 250 * time.Millisecond

// Timing constants for the network
const MASTER_BROADCAST_INTERVAL = 20 * time.Millisecond
const ELEV_STATE_TRANSMIT_INTERVAL = 20 * time.Millisecond

const HALL_ASSIGNMENT_ACK_TIMEOUT = 200 * time.Millisecond
const MASTER_CONNECTION_TIMEOUT = 1500 * time.Millisecond

const CONNECTION_REQ_INTERVAL = 20 * time.Millisecond
const STATE_TRANSITION_DECISION_INTERVAL = 3 * time.Second
const NODE_CONNECTION_TIMEOUT = 2 * time.Second
const PEER_POLL_INTERVAL = 20 * time.Millisecond
