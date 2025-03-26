package config

import (
	"time"
)

const DOOR_OPEN_DURATION = 3 * time.Second
const DOOR_STUCK_DURATION = 30 * time.Second
const NUM_FLOORS = 4
const NUM_BUTTONS = 3
const MSG_ID_PARTITION_SIZE = uint64(2 << 60)
const MASTER_TRANSMIT_INTERVAL = 50 * time.Millisecond
const ELEV_STATE_TRANSMIT_INTERVAL = 50 * time.Millisecond

const MASTER_CONNECTION_TIMEOUT = 500 * time.Millisecond

const ELEVATOR_STUCK_BETWEEN_FLOORS_TIMEOUT = 10 * time.Second
const ELEVATOR_STUCK_BETWEEN_FLOORS_POLL_INTERVAL = 1 * time.Second

const CONNECTION_REQ_INTERVAL = 100 * time.Millisecond
const DISCONNECTED_DECISION_INTERVAL = 2 * time.Second
const NODE_CONNECTION_TIMEOUT = 2 * time.Second
const PEER_POLL_INTERVAL = 20 * time.Millisecond
