package timer

import (
	"time"
)

var (
	timerEndTime time.Time
	timerActive  bool
)

func GetWallTime() time.Time {
	return time.Now()
}

func TimerStart(duration time.Duration) {
	timerEndTime = GetWallTime().Add(duration)
	timerActive = true
}

func TimerStop() {
	timerActive = false
}

func TimerTimedOut() bool {
	return timerActive && GetWallTime().After(timerEndTime)
}
