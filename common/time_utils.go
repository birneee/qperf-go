package common

import "time"

const (
	MinDuration time.Duration = -1 << 63
	MaxDuration time.Duration = 1<<63 - 1
)

func MaxTime(times []time.Time) time.Time {
	maxTime := time.Time{}
	for _, time := range times {
		if maxTime.Before(time) {
			maxTime = time
		}
	}
	return maxTime
}
