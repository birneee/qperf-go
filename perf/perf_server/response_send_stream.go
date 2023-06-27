package perf_server

import (
	"context"
	"github.com/quic-go/quic-go"
	"io"
	"qperf-go/common"
	"qperf-go/common/utils"
	"qperf-go/perf"
	"time"
)

type ResponseSendStream interface {
	StreamID() quic.StreamID
	Context() context.Context
	Length() uint64
	Delay() time.Duration
}

type responseSendStream struct {
	quicStream quic.SendStream
	// in bytes
	length     uint64
	delay      time.Duration
	connection *connection
}

func newResponseSendStream(quicStream quic.SendStream, length uint64, delay time.Duration, connection *connection) ResponseSendStream {
	s := &responseSendStream{
		quicStream: quicStream,
		length:     length,
		delay:      delay,
		connection: connection,
	}
	go func() {
		err := s.run()
		if err != nil {
			switch err := err.(type) {
			case *quic.StreamError:
				switch err.ErrorCode {
				case perf.DeadlineExceededStreamErrorCode:
				default:
					s.connection.close(err)
				}
			default:
				s.connection.close(err)
			}
		}
	}()
	return s
}

func (s *responseSendStream) run() error {
	time.Sleep(s.delay)
	var buf [65536]byte
	bytesToWrite := s.length
	_, err := io.CopyBuffer(s.quicStream, common.LimitReader(utils.InfiniteReader{}, bytesToWrite), buf[:])
	if err != nil {
		return err
	}
	_ = s.quicStream.Close()
	return nil
}

func (s *responseSendStream) StreamID() quic.StreamID {
	return s.quicStream.StreamID()
}

func (s *responseSendStream) Context() context.Context {
	return s.quicStream.Context()
}

func (s *responseSendStream) Length() uint64 {
	return s.length
}

func (s *responseSendStream) Delay() time.Duration {
	return s.delay
}
