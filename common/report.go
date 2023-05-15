package common

import (
	"github.com/quic-go/quic-go/logging"
	"time"
)

type Report struct {
	ReceivedBytes         logging.ByteCount
	ReceivedPackets       uint64
	TimeAggregated        time.Duration
	MinRTT                time.Duration
	MaxRTT                time.Duration
	SmoothedRTT           time.Duration
	PacketsLost           uint64
	SentBytes             logging.ByteCount
	ReceivedDatagramBytes logging.ByteCount
	SentDatagramBytes     logging.ByteCount
}
