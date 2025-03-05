package timer

import (
	"time"
)

type Timer struct {
	endTime time.Time
	active  bool
}

func Active(inTimer Timer) bool {
	return inTimer.active
}

func NewTimer() Timer {
	return Timer{endTime: time.Time{}, active: false}
}

func GetWallTime() time.Time {
	return time.Now()
}

func TimerStart(inTimer *Timer, duration time.Duration) {
	inTimer.endTime = GetWallTime().Add(duration)
	inTimer.active = true
}

func TimerStop(inTimer *Timer) {
	inTimer.active = false
}

func TimerTimedOut(inTimer Timer) bool {
	return inTimer.active && GetWallTime().After(inTimer.endTime)
}
