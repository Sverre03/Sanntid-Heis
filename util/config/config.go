package config

import (
	"time"
)

const DoorOpenDurationS = 3 * time.Second
const NumFloors = 4
const NumButtons = 3
const MsgIDSize = 2 << 12
const ConnectionTimeout = 500 * time.Millisecond
const MASTER_TRANSMIT_INTERVAL = 50 * time.Millisecond
const PortNum = 20011
