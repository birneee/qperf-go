package common

import (
	"context"
	"sync"
	"time"
)

type State struct {
	mutex                  sync.Mutex
	startCtx               context.Context
	startCtxCancel         context.CancelFunc
	startTime              time.Time
	establishmentCtx       context.Context
	establishmentCtxCancel context.CancelFunc
	establishmentTime      time.Time
	firstByteCtx           context.Context
	firstByteCtxCancel     context.CancelFunc
	firstByteTime          time.Time
	sentFirstByteCtx       context.Context
	sentFirstByteCtxCancel context.CancelFunc
	sentFirstByteTime      time.Time
	totalReceivedBytes     uint64
	totalReceivedPackets   uint64
	totalLostPackets       uint64
	totalSentBytes         uint64
	totalSentPackets       uint64
	congestionWindow       uint64
	maxStreamData          uint64
	bytesInFlight          uint64
}

func NewState() *State {
	s := &State{}
	s.startCtx, s.startCtxCancel = context.WithCancel(context.Background())
	s.establishmentCtx, s.establishmentCtxCancel = context.WithCancel(context.Background())
	s.firstByteCtx, s.firstByteCtxCancel = context.WithCancel(context.Background())
	s.sentFirstByteCtx, s.sentFirstByteCtxCancel = context.WithCancel(context.Background())
	return s
}

func (s *State) AddReceivedBytes(receivedBytes uint64) {
	s.mutex.Lock()
	s.totalReceivedBytes += receivedBytes
	if s.firstByteTime.IsZero() && s.totalReceivedBytes != 0 {
		print("blub")
		s.firstByteTime = time.Now()
		s.firstByteCtxCancel()
	}
	s.mutex.Unlock()
}

func (s *State) SetReceivedBytes(receivedBytes uint64) {
	s.mutex.Lock()
	if receivedBytes > s.totalReceivedBytes {
		s.totalReceivedBytes = receivedBytes
		if s.firstByteTime.IsZero() && s.totalReceivedBytes != 0 {
			select {
			case <-s.firstByteCtx.Done():
				// already set
			default:
				s.firstByteTime = time.Now()
				s.firstByteCtxCancel()
			}
		}
	}
	s.mutex.Unlock()
}

func (s *State) SetSentBytes(sentBytes uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if sentBytes > s.totalSentBytes {
		s.totalSentBytes = sentBytes
		select {
		case <-s.sentFirstByteCtx.Done():
			// already set
		default:
			s.sentFirstByteTime = time.Now()
			s.sentFirstByteCtxCancel()
		}
	}
}

func (s *State) AddReceivedPackets(receivedPackets uint64) {
	s.mutex.Lock()
	s.totalReceivedPackets += receivedPackets
	s.mutex.Unlock()
}

func (s *State) Total() (receivedBytes uint64, receivedPackets uint64) {
	s.mutex.Lock()
	receivedBytes = s.totalReceivedBytes
	receivedPackets = s.totalReceivedPackets
	s.mutex.Unlock()
	return
}

// StartTime blocks until a value is available
func (s *State) StartTime() time.Time {
	<-s.startCtx.Done()
	return s.startTime
}

func (s *State) SetStartTime() {
	select {
	case <-s.startCtx.Done():
		// already set
	default:
		s.startTime = time.Now()
		s.startCtxCancel()
	}
}

// FirstByteTime blocks until a value is available
func (s *State) FirstByteTime() time.Time {
	<-s.firstByteCtx.Done()
	return s.firstByteTime
}

// SentFirstByteTime blocks until a value is available
func (s *State) SentFirstByteTime() time.Time {
	<-s.sentFirstByteCtx.Done()
	return s.sentFirstByteTime
}

// EstablishmentTime blocks until a value is available
func (s *State) EstablishmentTime() time.Time {
	<-s.establishmentCtx.Done()
	return s.establishmentTime
}

func (s *State) SetEstablishmentTime() {
	select {
	case <-s.establishmentCtx.Done():
		// already set
	default:
		s.establishmentTime = time.Now()
		s.establishmentCtxCancel()
	}
}

func (s *State) IncrementLostPackets() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.totalLostPackets++
}

func (s *State) IncrementSentPackets() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.totalSentPackets++
}

func (s *State) LostPackets() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.totalLostPackets
}

func (s *State) ReceivedBytes() (receivedBytes uint64) {
	s.mutex.Lock()
	receivedBytes = s.totalReceivedBytes
	s.mutex.Unlock()
	return
}

func (s *State) ReceivedPackets() (receivedPackets uint64) {
	s.mutex.Lock()
	receivedPackets = s.totalReceivedPackets
	s.mutex.Unlock()
	return
}

func (s *State) SentBytes() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.totalSentBytes
}

func (s *State) SentPackets() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.totalSentPackets
}

func (s *State) IncrementReceivedPackets() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.totalReceivedPackets++
}

func (s *State) SetCongestionWindow(value uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.congestionWindow = value
}

func (s *State) SetMaxStreamData(value uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.maxStreamData = value
}

func (s *State) SetBytesInFlight(value uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.bytesInFlight = value
}

func (s *State) CongestionWindow() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.congestionWindow
}

func (s *State) FreeReceiveWindow() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.maxStreamData < s.totalSentBytes {
		return 0
	} else {
		return s.maxStreamData - s.totalSentBytes
	}
}

func (s *State) FreeCongestionWindow() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.congestionWindow < s.bytesInFlight {
		return 0
	} else {
		return s.congestionWindow - s.bytesInFlight
	}
}
