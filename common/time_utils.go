package common

import "time"

func MaxTime(times []time.Time) time.Time {
	maxTime := time.Time{}
	for _, time := range times {
		if maxTime.Before(time) {
			maxTime = time
		}
	}
	return maxTime
}
