package server

import (
	"context"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"net"
	"qperf-go/common"
	"sync"
	"time"
)

type qperfServerSession struct {
	session   quic.Session
	sessionID uint64
	// used to detect migration
	currentRemoteAddr net.Addr
	logger            common.Logger
	closeOnce         sync.Once
	state             *common.State
}

func (s *qperfServerSession) run() {
	s.logger.Infof("open")
	if s.session.ExtraStreamEncrypted() {
		s.logger.Infof("use XSE-QUIC")
	}

	go s.reportLoop()

	for {
		quicStream, err := s.session.AcceptStream(context.Background())
		if err != nil {
			s.close(err)
			return
		}

		qperfStream := &qperfServerStream{
			session: s,
			stream:  quicStream,
			logger:  s.logger.WithPrefix(fmt.Sprintf("stream %d", quicStream.StreamID())),
		}

		go qperfStream.run()
	}
}

func (s *qperfServerSession) reportLoop() {
	lastTime := s.state.SentFirstByteTime()
	lastLostPackets := uint64(0)
	lastBytes := uint64(0)
	lastPackets := uint64(0)
	for {
		select {
		case <-s.session.Context().Done():
			// session is closed
			s.logger.Infof("total: bytes sent: %d B, packets sent: %d, packets lost: %d",
				s.state.ReceivedBytes(),
				s.state.ReceivedPackets(),
				s.state.LostPackets(),
			)
			return
		default:
			// session is open
		}
		time.Sleep(time.Second)
		time := time.Now()
		packetLost := s.state.LostPackets()
		bytes := s.state.SentBytes()
		packets := s.state.SentPackets()
		s.logger.Infof("second %f: %f bit/s, cwnd %d B, free cwnd %d B, free rwnd %d B, bytes sent: %d B, packets sent: %d, packets lost: %d",
			time.Sub(s.state.FirstByteTime()).Seconds(),
			float64(bytes-lastBytes)*8/time.Sub(lastTime).Seconds(),
			s.state.CongestionWindow(),
			s.state.FreeCongestionWindow(),
			s.state.FreeReceiveWindow(),
			bytes-lastBytes,
			packets-lastPackets,
			packetLost-lastLostPackets,
		)
		lastTime = time
		lastLostPackets = packetLost
		lastBytes = bytes
		lastPackets = packets
	}
}

func (s *qperfServerSession) checkIfRemoteAddrChanged() {
	if s.currentRemoteAddr != s.session.RemoteAddr() {
		s.currentRemoteAddr = s.session.RemoteAddr()
		s.logger.Infof("migrated to %s", s.currentRemoteAddr)
	}
}

func (s *qperfServerSession) close(err error) {
	s.closeOnce.Do(func() {
		switch err := err.(type) {
		case *quic.ApplicationError:
			if err.ErrorCode == common.RuntimeReachedErrorCode {
				s.logger.Infof("close")
			} else {
				s.logger.Errorf("close with error: %s", err)
			}
		default:
			s.logger.Errorf("close with error: %s", err)
		}
	})
}
