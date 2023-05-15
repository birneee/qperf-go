package common

import (
	"context"
	"github.com/quic-go/quic-go/logging"
	"sync"
	"time"
)

type State struct {
	mutex                      sync.Mutex
	startTime                  time.Time
	firstByteTime              time.Time
	establishmentTime          time.Time
	handshakeConfirmedTime     time.Time
	totalReceivedBytes         uint64
	totalReceivedPackets       uint64
	totalMinRTT                time.Duration
	totalMaxRTT                time.Duration
	totalPacketsLost           uint64
	totalSentBytes             uint64
	totalReceivedDatagramBytes logging.ByteCount
	totalSentDatagramBytes     logging.ByteCount
	// contexts
	handshakeConfirmedCtx    context.Context
	handshakeConfirmedCancel context.CancelFunc
	firstByteReceivedCtx     context.Context
	firstByteReceivedCancel  context.CancelFunc
	// values below are reset
	lastReportTime                time.Time
	lastReportReceivedBytes       uint64
	lastReportReceivedPackets     uint64
	minRTT                        time.Duration
	maxRTT                        time.Duration
	smoothedRTT                   time.Duration
	packetsLost                   uint64
	lastReportSentBytes           uint64
	intervalReceivedDatagramBytes logging.ByteCount
	intervalSentDatagramBytes     logging.ByteCount
}

func NewState() *State {
	s := &State{}
	s.handshakeConfirmedCtx, s.handshakeConfirmedCancel = context.WithCancel(context.Background())
	s.firstByteReceivedCtx, s.firstByteReceivedCancel = context.WithCancel(context.Background())
	return s
}

func (s *State) AddReceivedBytes(receivedBytes uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.totalReceivedBytes += receivedBytes
}

func (s *State) AddReceivedPackets(receivedPackets uint64) {
	s.mutex.Lock()
	s.totalReceivedPackets += receivedPackets
	s.mutex.Unlock()
}

func (s *State) GetAndResetReport() Report {
	now := time.Now()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	report := Report{
		ReceivedBytes:         logging.ByteCount(s.totalReceivedBytes - s.lastReportReceivedBytes),
		ReceivedPackets:       s.totalReceivedPackets - s.lastReportReceivedPackets,
		TimeAggregated:        now.Sub(MaxTime([]time.Time{s.lastReportTime, s.startTime})),
		MinRTT:                s.minRTT,
		MaxRTT:                s.maxRTT,
		SmoothedRTT:           s.smoothedRTT,
		PacketsLost:           s.packetsLost,
		SentBytes:             logging.ByteCount(s.totalSentBytes - s.lastReportSentBytes),
		ReceivedDatagramBytes: s.intervalReceivedDatagramBytes,
		SentDatagramBytes:     s.intervalSentDatagramBytes,
	}
	// reset
	s.lastReportTime = now
	s.lastReportReceivedBytes = s.totalReceivedBytes
	s.lastReportReceivedPackets = s.totalReceivedPackets
	s.lastReportSentBytes = s.totalSentBytes
	s.minRTT = MaxDuration
	s.maxRTT = MinDuration
	s.smoothedRTT = -1
	s.packetsLost = 0
	s.intervalReceivedDatagramBytes = 0
	s.intervalSentDatagramBytes = 0
	return report
}

func (s *State) TotalReport() Report {
	now := time.Now()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	report := Report{
		ReceivedBytes:         logging.ByteCount(s.totalReceivedBytes),
		ReceivedPackets:       s.totalReceivedPackets,
		TimeAggregated:        now.Sub(s.startTime),
		MinRTT:                s.totalMinRTT,
		MaxRTT:                s.totalMaxRTT,
		PacketsLost:           s.totalPacketsLost,
		SentBytes:             logging.ByteCount(s.totalSentBytes),
		ReceivedDatagramBytes: s.totalReceivedDatagramBytes,
		SentDatagramBytes:     s.totalSentDatagramBytes,
	}
	return report
}

func (s *State) Total() (receivedBytes uint64, receivedPackets uint64) {
	s.mutex.Lock()
	receivedBytes = s.totalReceivedBytes
	receivedPackets = s.totalReceivedPackets
	s.mutex.Unlock()
	return
}

func (s *State) StartTime() time.Time {
	return s.startTime
}

func (s *State) SetStartTime() {
	if !s.startTime.IsZero() {
		panic("already set")
	}
	s.startTime = time.Now()
}

func (s *State) FirstByteTime() time.Time {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.firstByteTime
}

func (s *State) SetEstablishmentTime() {
	if !s.establishmentTime.IsZero() {
		panic("already set")
	}
	s.establishmentTime = time.Now()
}

func (s *State) EstablishmentTime() time.Time {
	return s.establishmentTime
}

func (s *State) GetStartTime() time.Time {
	return s.startTime
}

func (s *State) AddRttStats(stats *logging.RTTStats) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.minRTT = Min(stats.LatestRTT(), s.minRTT)
	s.totalMinRTT = Min(stats.LatestRTT(), s.totalMinRTT)
	s.maxRTT = Max(stats.LatestRTT(), s.maxRTT)
	s.totalMaxRTT = Max(stats.LatestRTT(), s.totalMaxRTT)
	s.smoothedRTT = stats.SmoothedRTT()
}

func (s *State) MinRTT() time.Duration {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.minRTT
}

func (s *State) MaxRTT() time.Duration {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.maxRTT
}

func (s *State) SmoothedRTT() time.Duration {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.smoothedRTT
}

func (s *State) AddLostPackets(n uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.packetsLost += n
	s.totalPacketsLost += n
}

func (s *State) PacketsLost() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.packetsLost
}

func (s *State) SetHandshakeConfirmedTime() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.handshakeConfirmedTime = time.Now()
}

func (s *State) AwaitHandshakeConfirmed() {
	<-s.handshakeConfirmedCtx.Done()
}

func (s *State) AwaitFirstByteReceived() {
	<-s.firstByteReceivedCtx.Done()
}

func (s *State) HandshakeConfirmedTime() time.Time {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.handshakeConfirmedTime
}

func (s *State) AddSentBytes(u uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.totalSentBytes += u
}

func (s *State) MaybeSetFirstByteReceived() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.maybeSetFirstByteReceived()
}

// must only be called while holding the lock
func (s *State) maybeSetFirstByteReceived() {
	if s.firstByteTime.IsZero() {
		s.firstByteTime = time.Now()
		s.firstByteReceivedCancel()
	}
}

func (s *State) AddReceivedDatagramBytes(length logging.ByteCount) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.totalReceivedDatagramBytes += length
	s.intervalReceivedDatagramBytes += length
	s.maybeSetFirstByteReceived()
}

func (s *State) AddSentDatagramBytes(length logging.ByteCount) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.totalSentDatagramBytes += length
	s.intervalSentDatagramBytes += length
}
