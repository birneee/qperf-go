package testutils

import (
	"context"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
)

type ZeroRttTracer interface {
	NewConnectionTracer(ctx context.Context, perspective logging.Perspective, id quic.ConnectionID) *logging.ConnectionTracer
	FirstByteChan() chan struct{}
	ReceivedBytes() int
}

type zeroRttTracer struct {
	receivedBytes int
	firstByteChan chan struct{}
}

func (t *zeroRttTracer) ReceivedBytes() int {
	return t.receivedBytes
}

func NewZeroRttTracer() ZeroRttTracer {
	return &zeroRttTracer{
		firstByteChan: make(chan struct{}),
	}
}

func (t *zeroRttTracer) NewConnectionTracer(_ctx context.Context, _perspective logging.Perspective, _id quic.ConnectionID) *logging.ConnectionTracer {
	return &logging.ConnectionTracer{
		ReceivedLongHeaderPacket: func(_ *logging.ExtendedHeader, _ logging.ByteCount, ecn logging.ECN, frames []logging.Frame) {
			for _, frame := range frames {
				switch frame := frame.(type) {
				case *logging.StreamFrame:
					select {
					case <-t.firstByteChan:
					default:
						close(t.firstByteChan)
					}
					t.receivedBytes = int(frame.Offset + frame.Length)
				}
			}
		},
	}
}

func (t *zeroRttTracer) FirstByteChan() chan struct{} {
	return t.firstByteChan
}
