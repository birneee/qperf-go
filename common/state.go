package common

import (
	"sync"
	"time"
)

type State struct {
	mutex                     sync.Mutex
	startTime                 time.Time
	firstByteTime             time.Time
	totalReceivedBytes        uint64
	totalReceivedPackets      uint64
	lastReportTime            time.Time
	lastReportReceivedBytes   uint64
	lastReportReceivedPackets uint64
}

func (s *State) Add(receivedBytes uint64, receivedPackets uint64) {
	s.mutex.Lock()
	s.totalReceivedBytes += receivedBytes
	s.totalReceivedPackets += receivedPackets
	if s.firstByteTime.IsZero() && s.totalReceivedBytes != 0 {
		s.firstByteTime = time.Now()
	}
	s.mutex.Unlock()
}

// GetAndResetReport
//
// returns (receivedBytes, receivedPackets, delta)
func (s *State) GetAndResetReport() (uint64, uint64, time.Duration) {
	now := time.Now()
	s.mutex.Lock()
	receivedBytes := s.totalReceivedBytes - s.lastReportReceivedBytes
	receivedPackets := s.totalReceivedPackets - s.lastReportReceivedPackets
	delta := now.Sub(MaxTime([]time.Time{s.lastReportTime, s.firstByteTime, s.startTime}))
	s.lastReportTime = now
	s.lastReportReceivedBytes = s.totalReceivedBytes
	s.lastReportReceivedPackets = s.totalReceivedPackets
	s.mutex.Unlock()
	return receivedBytes, receivedPackets, delta
}

// Total
//
// returns (receivedBytes, receivedPackets)
func (s *State) Total() (uint64, uint64) {
	s.mutex.Lock()
	receivedBytes := s.totalReceivedBytes
	receivedPackets := s.totalReceivedPackets
	s.mutex.Unlock()
	return receivedBytes, receivedPackets
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

func (s *State) GetOrWaitFirstByteTime() time.Time {
	for {
		s.mutex.Lock()
		value := s.firstByteTime
		s.mutex.Unlock()
		if !value.IsZero() {
			return value
		}
		time.Sleep(1 * time.Millisecond)
	}
}
