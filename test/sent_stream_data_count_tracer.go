package test

import "github.com/quic-go/quic-go/logging"

type SentStreamDataCountTracer struct {
	logging.NullConnectionTracer
	Count logging.ByteCount
}

func (n *SentStreamDataCountTracer) SentShortHeaderPacket(_ *logging.ShortHeader, _ logging.ByteCount, _ *logging.AckFrame, frames []logging.Frame) {
	for _, frame := range frames {
		switch frame := frame.(type) {
		case *logging.StreamFrame:
			n.Count = frame.Offset + frame.Length
		}
	}
}
