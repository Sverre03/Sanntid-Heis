package timer

import (
	"time"
)

type Timer struct {
	EndTime time.Time
	Active  bool
}

func Active(inTimer Timer) bool {
	return inTimer.Active
}

func NewTimer() Timer {
	return Timer{EndTime: time.Time{}, Active: false}
}

func GetWallTime() time.Time {
	return time.Now()
}

func TimerStart(inTimer *Timer, duration time.Duration) {
	inTimer.EndTime = GetWallTime().Add(duration)
	inTimer.Active = true
}

func TimerStop(inTimer *Timer) {
	inTimer.Active = false
}

func TimerTimedOut(inTimer Timer) bool {
	return inTimer.Active && GetWallTime().After(inTimer.EndTime)
}
