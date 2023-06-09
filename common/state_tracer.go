package common

import (
	"context"
	"github.com/quic-go/quic-go/logging"
	"time"
)

type StateTracer struct {
	logging.NullTracer
	State *State
}

func (a StateTracer) TracerForConnection(ctx context.Context, p logging.Perspective, odcid logging.ConnectionID) logging.ConnectionTracer {
	return StateConnectionTracer{
		State: a.State,
	}
}

type StateConnectionTracer struct {
	logging.NullConnectionTracer
	State *State
}

func NewStateTracer(state *State) func(ctx context.Context, perspective logging.Perspective, connectionID logging.ConnectionID) logging.ConnectionTracer {
	return func(ctx context.Context, perspective logging.Perspective, connectionID logging.ConnectionID) logging.ConnectionTracer {
		return StateConnectionTracer{
			State: state,
		}
	}
}

func (n StateConnectionTracer) ReceivedLongHeaderPacket(*logging.ExtendedHeader, logging.ByteCount, []logging.Frame) {
	n.State.AddReceivedPackets(1)
}

func (n StateConnectionTracer) ReceivedShortHeaderPacket(header *logging.ShortHeader, bytes logging.ByteCount, frames []logging.Frame) {
	n.State.AddReceivedPackets(1)
	for _, frame := range frames {
		switch frame := frame.(type) {
		case *logging.HandshakeDoneFrame:
			n.State.SetHandshakeConfirmedTime()
			n.State.handshakeConfirmedCancel()
		case *logging.StreamFrame:
			if frame.Offset == 0 {
				n.State.MaybeSetFirstByteReceived()
			}
		case *logging.DatagramFrame:
			n.State.AddReceivedDatagramBytes(frame.Length)
		}
	}
}

func (n StateConnectionTracer) SentLongHeaderPacket(_ *logging.ExtendedHeader, _ logging.ByteCount, _ *logging.AckFrame, frames []logging.Frame) {
	for _, frame := range frames {
		switch frame := frame.(type) {
		case *logging.StreamFrame:
			if frame.Offset == 0 {
				n.State.MaybeSetFirstByteSent()
			}
		case *logging.DatagramFrame:
			n.State.AddSentDatagramBytes(frame.Length)
		}
	}
}

func (n StateConnectionTracer) SentShortHeaderPacket(_ *logging.ShortHeader, _ logging.ByteCount, _ *logging.AckFrame, frames []logging.Frame) {
	for _, frame := range frames {
		switch frame := frame.(type) {
		case *logging.StreamFrame:
			if frame.Offset == 0 {
				n.State.MaybeSetFirstByteSent()
			}
		case *logging.DatagramFrame:
			n.State.AddSentDatagramBytes(frame.Length)
		}
	}
}

func (n StateConnectionTracer) UpdatedMetrics(rttStats *logging.RTTStats, cwnd, bytesInFlight logging.ByteCount, packetsInFlight int) {
	n.State.AddRttStats(rttStats)
}

func (n StateConnectionTracer) LostPacket(encLevel logging.EncryptionLevel, pn logging.PacketNumber, reason logging.PacketLossReason) {
	n.State.AddLostPackets(1)
}

func (n StateConnectionTracer) UpdatedKeyFromTLS(encLevel logging.EncryptionLevel, _ logging.Perspective) {
	if encLevel == logging.Encryption1RTT {
		now := time.Now()
		n.State.SetHandshakeCompletedTime(now)
	}
}
