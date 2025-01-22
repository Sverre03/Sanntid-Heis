package time

import (
    "time"
)

var (
    timerEndTime time.Time
    timerActive  bool
)

func getWallTime() time.Time {
    return time.Now()
}

func timerStart(duration time.Duration) {
    timerEndTime = getWallTime().Add(duration)
    timerActive = true
}

func timerStop() {
    timerActive = false
}

func timerTimedOut() bool {
    return timerActive && getWallTime().After(timerEndTime)
}
