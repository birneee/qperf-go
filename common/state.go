package common

import (
	"sync"
	"time"
)

type State struct {
	mutex                     sync.Mutex
	startTime                 time.Time
	firstByteTime             time.Time
	establishmentTime         time.Time
	totalReceivedBytes        uint64
	totalReceivedPackets      uint64
	lastReportTime            time.Time
	lastReportReceivedBytes   uint64
	lastReportReceivedPackets uint64
}

func (s *State) AddReceivedBytes(receivedBytes uint64) {
	s.mutex.Lock()
	s.totalReceivedBytes += receivedBytes
	if s.firstByteTime.IsZero() && s.totalReceivedBytes != 0 {
		s.firstByteTime = time.Now()
	}
	s.mutex.Unlock()
}

func (s *State) AddReceivedPackets(receivedPackets uint64) {
	s.mutex.Lock()
	s.totalReceivedPackets += receivedPackets
	s.mutex.Unlock()
}

func (s *State) GetAndResetReport() (receivedBytes uint64, receivedPackets uint64, delta time.Duration) {
	now := time.Now()
	s.mutex.Lock()
	receivedBytes = s.totalReceivedBytes - s.lastReportReceivedBytes
	receivedPackets = s.totalReceivedPackets - s.lastReportReceivedPackets
	delta = now.Sub(MaxTime([]time.Time{s.lastReportTime, s.firstByteTime, s.startTime}))
	s.lastReportTime = now
	s.lastReportReceivedBytes = s.totalReceivedBytes
	s.lastReportReceivedPackets = s.totalReceivedPackets
	s.mutex.Unlock()
	return
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

func (s *State) GetFirstByteTime() time.Time {
	s.mutex.Lock()
	value := s.firstByteTime
	s.mutex.Unlock()
	if value.IsZero() {
		panic("not set yet")
	}
	return value
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
