package perf_client

import (
	"context"
	"github.com/quic-go/quic-go"
	"io"
	"qperf-go/common/utils"
	"qperf-go/perf"
	"sync/atomic"
)

type ResponseReceiveStream interface {
	ReceivedBytes() uint64
	Context() context.Context
	Cancel()
	Success() bool
}

type responseReceiveStream struct {
	receivedBytes atomic.Uint64
	client        *client
	quicStream    quic.ReceiveStream
	ctx           context.Context
	cancelCtx     context.CancelFunc
	success       bool
}

func newResponseReceiveStream(quicStream quic.ReceiveStream, client *client) (ResponseReceiveStream, error) {
	s := &responseReceiveStream{
		client:     client,
		quicStream: quicStream,
	}
	s.ctx, s.cancelCtx = context.WithCancel(client.Context())
	go func() {
		err := s.run()
		if err != nil {
			switch err := err.(type) {
			case *quic.StreamError:
				switch err.ErrorCode {
				case perf.DeadlineExceededStreamErrorCode:
				default:
					s.client.close(err)
				}
			default:
				s.client.close(err)
			}
		}
	}()
	return s, nil
}

func (s *responseReceiveStream) ReceivedBytes() uint64 {
	return s.receivedBytes.Load()
}

func (s *responseReceiveStream) run() error {
	var buf [65536]byte
	_, err := io.CopyBuffer(utils.FuncToWriter(func(p []byte) (n int, err error) {
		s.receivedBytes.Add(uint64(len(p)))
		s.client.receivedBytes.Add(uint64(len(p)))
		return len(p), nil
	}), s.quicStream, buf[:])
	if err != nil {
		return err
	}
	s.success = true
	s.cancelCtx()
	return nil
}

func (s *responseReceiveStream) Context() context.Context {
	return s.ctx
}

func (s *responseReceiveStream) Cancel() {
	s.quicStream.CancelRead(perf.DeadlineExceededStreamErrorCode)
}

func (s *responseReceiveStream) Success() bool {
	return s.success
}
