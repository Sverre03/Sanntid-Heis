package util

import "elev/config"

func MapIsEmpty[k comparable, v any](m map[k]v) bool {
	return len(m) == 0
}

func KeyExistsInMap[K comparable, V any](key K, m map[K]V) bool {
	_, ok := m[key]
	return ok
}

func HallAssignmentIsRemoved(oldGlobalHallRequests [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool,
	newGlobalHallReq [config.NUM_FLOORS][config.NUM_HALL_BUTTONS]bool) bool {
	for floor := range config.NUM_FLOORS {
		for button := range 2 {
			// If change is from (true -> false) => Assignment completed
			if oldGlobalHallRequests[floor][button] && !newGlobalHallReq[floor][button] {
				return true
			}
		}
	}
	return false
}

func IncrementIntCounter(counter int) int {
	counter += 1
	if counter < 0 {
		counter = 1
	}
	return counter
}

func IncrementCounterUint64(counter uint64) uint64 {
	counter += 1
	if counter == 0 {
		counter = 1
	}
	return counter
}

func MyCounterIsSmaller(myCounter uint64, otherCounter uint64) bool {
	return myCounter < otherCounter
	// halfRange := MAX_UINT64_COUNTER_VALUE / 2
	// return ((myCounter - otherCounter) % MAX_UINT64_COUNTER_VALUE) < halfRange
}
