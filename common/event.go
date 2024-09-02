package common

import (
	"time"
)

type Event interface {
	Time() time.Duration
}
