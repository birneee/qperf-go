package perf_server

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/quic-go/quic-go"
	"io"
	"qperf-go/perf"
	"sync"
	"sync/atomic"
	"time"
)

type RequestReceiveStream interface {
	ResponseLength() uint64
	// Context of stream is done on successful reception or error
	Context() context.Context
	// Success returns true if full request is received
	Success() bool
	StreamID() quic.StreamID
	ResponseDelay() time.Duration
}

type requestReceiveStream struct {
	quicStream     quic.ReceiveStream
	connection     *connection
	responseLength uint64
	responseDelay  time.Duration
	ctx            context.Context
	ctxCancel      context.CancelFunc
	receivedBytes  atomic.Uint64
	success        bool
	closeOnce      sync.Once
	err            error
}

func newRequestReceiveStream(quicStream quic.ReceiveStream, connection *connection) (RequestReceiveStream, error) {
	s := &requestReceiveStream{
		quicStream: quicStream,
		connection: connection,
	}
	s.ctx, s.ctxCancel = context.WithCancel(connection.Context())
	go func() {
		err := s.run()
		if err != nil {
			s.close(err)
		}
	}()
	return s, nil
}

func (s *requestReceiveStream) run() error {
	var buf [65536]byte
	n, err := s.quicStream.Read(buf[:12])
	if err != nil && err != io.EOF {
		return err
	}
	if n != 12 {
		return fmt.Errorf("invalid request header")
	}
	s.responseLength = binary.LittleEndian.Uint64(buf[0:8])
	s.responseDelay = time.Duration(binary.LittleEndian.Uint32(buf[8:12])) * time.Millisecond
	s.receivedBytes.Add(uint64(n))

	for {
		n, err := s.quicStream.Read(buf[:])
		s.receivedBytes.Add(uint64(n))
		if err != nil {
			if err == io.EOF {
				break
			} else {
				s.ctxCancel()
				return err
			}
		}
	}
	s.success = true
	s.ctxCancel()
	return nil
}

func (s *requestReceiveStream) ResponseLength() uint64 {
	return s.responseLength
}

func (s *requestReceiveStream) ResponseDelay() time.Duration {
	return s.responseDelay
}

func (s *requestReceiveStream) Context() context.Context {
	return s.ctx
}

func (s *requestReceiveStream) Success() bool {
	return s.success
}

func (s *requestReceiveStream) StreamID() quic.StreamID {
	return s.quicStream.StreamID()
}

func (s *requestReceiveStream) close(err error) {
	s.closeOnce.Do(func() {
		s.err = err
		s.closeOnce.Do(func() {
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
		})
	})
}
