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
			// Check if change is from (true -> false), assignment complete
			if oldGlobalHallRequests[floor][button] && !newGlobalHallReq[floor][button] {
				// fmt.Printf("Hall assignment removed at floor %d, button %d\n", floor, button)
				return true
			}
		}
	}
	return false
}
