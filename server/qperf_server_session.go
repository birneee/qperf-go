package server

import (
	"context"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"net"
	"qperf-go/common"
)

type qperfServerSession struct {
	session   quic.Session
	sessionID uint64
	// used to detect migration
	currentRemoteAddr net.Addr
	logger            common.Logger
}

func (s *qperfServerSession) run() {
	s.logger.Infof("open")

	for {
		quicStream, err := s.session.AcceptStream(context.Background())
		if err != nil {
			s.logger.Errorf("%s", err)
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

func (s *qperfServerSession) checkIfRemoteAddrChanged() {
	if s.currentRemoteAddr != s.session.RemoteAddr() {
		s.currentRemoteAddr = s.session.RemoteAddr()
		s.logger.Infof("migrated to %s", s.currentRemoteAddr)
	}
}
