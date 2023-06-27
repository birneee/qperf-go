package perf_client

import (
	"encoding/binary"
	"github.com/quic-go/quic-go"
	"io"
	"qperf-go/common"
	"qperf-go/common/utils"
	"qperf-go/perf"
	"sync"
	"sync/atomic"
	"time"
)

type RequestSendStream interface {
	SentBytes() uint64
	Cancel()
}

type requestSendStream struct {
	quicStream quic.SendStream
	// additional to the 8 bytes of response length header
	requestLength  uint64
	responseLength uint64
	responseDelay  time.Duration
	sentBytes      atomic.Uint64
	client         *client
	closeOnce      sync.Once
	err            error
}

func newRequestSendStream(quicStream quic.SendStream, requestLength uint64, responseLength uint64, responseDelay time.Duration, client *client) RequestSendStream {
	s := &requestSendStream{
		quicStream:     quicStream,
		requestLength:  requestLength,
		responseLength: responseLength,
		responseDelay:  responseDelay,
		client:         client,
	}
	go func() {
		err := s.run()
		if err != nil {
			s.close(err)
			return
		}
	}()
	return s
}

func (s *requestSendStream) run() error {
	var buf [65536]byte
	binary.LittleEndian.PutUint64(buf[:], s.responseLength)
	binary.LittleEndian.PutUint32(buf[8:], uint32(s.responseDelay.Milliseconds()))
	sendStream := io.MultiWriter(s.quicStream, utils.FuncToWriter(func(p []byte) (n int, err error) {
		s.sentBytes.Add(uint64(len(p)))
		s.client.sentBytes.Add(uint64(len(p)))
		return len(p), err
	}))
	_, err := sendStream.Write(buf[:12])
	if err != nil {
		return err
	}
	_, err = io.CopyBuffer(sendStream, common.LimitReader(utils.InfiniteReader{}, s.requestLength), buf[:])
	if err != nil {
		return err
	}
	err = s.quicStream.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *requestSendStream) SentBytes() uint64 {
	return s.sentBytes.Load()
}

func (s *requestSendStream) Cancel() {
	s.quicStream.CancelWrite(perf.DeadlineExceededStreamErrorCode)
	s.close(&quic.StreamError{
		StreamID:  s.quicStream.StreamID(),
		ErrorCode: perf.DeadlineExceededStreamErrorCode,
		Remote:    false,
	})
}

func (s *requestSendStream) close(err error) {
	s.closeOnce.Do(func() {
		s.err = err
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
	})
}
