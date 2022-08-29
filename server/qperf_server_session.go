package server

import (
	"context"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"net"
	"qperf-go/common"
	"sync"
)

type qperfServerSession struct {
	connection   quic.Connection
	connectionID uint64
	// used to detect migration
	currentRemoteAddr net.Addr
	logger            common.Logger
	closeOnce         sync.Once
}

func (s *qperfServerSession) run() {
	s.logger.Infof("open")
	if s.session.ExtraStreamEncrypted() {
		s.logger.Infof("use XSE-QUIC")
	}

	for {
		quicStream, err := s.connection.AcceptStream(context.Background())
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

func (s *qperfServerSession) checkIfRemoteAddrChanged() {
	if s.currentRemoteAddr != s.connection.RemoteAddr() {
		s.currentRemoteAddr = s.connection.RemoteAddr()
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
