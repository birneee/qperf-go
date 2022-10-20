package server

import (
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"io"
	"qperf-go/common"
)

type qperfServerStream struct {
	session *qperfServerSession
	stream  quic.Stream
	logger  common.Logger
}

func (s *qperfServerStream) run() {
	s.logger.Infof("open")

	request, err := io.ReadAll(s.stream)
	if err != nil {
		s.session.close(err)
		s.logger.Errorf("%s", err)
		return
	}
	if string(request) != common.QPerfStartSendingRequest {
		s.session.close(fmt.Errorf("unknown qperf message"))
		return
	}

	buf := make([]byte, 65536)
	for {
		_, err := s.stream.Write(buf)
		if err != nil {
			s.session.close(err)
			return
		}
		s.session.checkIfRemoteAddrChanged()
	}
}
