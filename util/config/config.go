package config

import (
	"time"
)

const DOOR_OPEN_DURATION = 3 * time.Second
const NUM_FLOORS = 4
const NUM_BUTTONS = 3
const MSG_ID_PARTITION_SIZE = 2 << 16
const CONNECTION_TIMEOUT = 500 * time.Millisecond
const MASTER_TRANSMIT_INTERVAL = 50 * time.Millisecond
const ELEV_STATE_TRANSMIT_INTERVAL = 50 * time.Millisecond
const PORT_NUM = 20011
const INPUT_POLL_RATE = 25 * time.Millisecond
