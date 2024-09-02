package common

import (
	"context"
	"github.com/quic-go/quic-go/logging"
	"time"
)

type StateTracer struct {
	State *State
}

func (t StateTracer) TracerForConnection(_ context.Context, _ logging.Perspective, _ logging.ConnectionID) *logging.ConnectionTracer {
	return &logging.ConnectionTracer{
		ReceivedLongHeaderPacket: func(_ *logging.ExtendedHeader, _ logging.ByteCount, _ logging.ECN, _ []logging.Frame) {
			t.State.AddReceivedPackets(1)
		},
		ReceivedShortHeaderPacket: func(_ *logging.ShortHeader, _ logging.ByteCount, _ logging.ECN, frames []logging.Frame) {
			t.State.AddReceivedPackets(1)
			for _, frame := range frames {
				switch frame := frame.(type) {
				case *logging.HandshakeDoneFrame:
					t.State.SetHandshakeConfirmedTime()
					t.State.handshakeConfirmedCancel()
				case *logging.StreamFrame:
					if frame.Offset == 0 {
						t.State.MaybeSetFirstByteReceived()
					}
				case *logging.DatagramFrame:
					t.State.AddReceivedDatagramBytes(frame.Length)
				}
			}
		},
		SentLongHeaderPacket: func(header *logging.ExtendedHeader, count logging.ByteCount, ecn logging.ECN, frame *logging.AckFrame, frames []logging.Frame) {
			for _, frame := range frames {
				switch frame := frame.(type) {
				case *logging.StreamFrame:
					if frame.Offset == 0 {
						t.State.MaybeSetFirstByteSent()
					}
				case *logging.DatagramFrame:
					t.State.AddSentDatagramBytes(frame.Length)
				}
			}
		},
		SentShortHeaderPacket: func(header *logging.ShortHeader, count logging.ByteCount, ecn logging.ECN, frame *logging.AckFrame, frames []logging.Frame) {
			for _, frame := range frames {
				switch frame := frame.(type) {
				case *logging.StreamFrame:
					if frame.Offset == 0 {
						t.State.MaybeSetFirstByteSent()
					}
				case *logging.DatagramFrame:
					t.State.AddSentDatagramBytes(frame.Length)
				}
			}
		},
		UpdatedMetrics: func(rttStats *logging.RTTStats, cwnd, bytesInFlight logging.ByteCount, packetsInFlight int) {
			t.State.AddRttStats(rttStats)

		},
		LostPacket: func(level logging.EncryptionLevel, number logging.PacketNumber, reason logging.PacketLossReason) {
			t.State.AddLostPackets(1)

		},
		UpdatedKeyFromTLS: func(level logging.EncryptionLevel, perspective logging.Perspective) {
			if level == logging.Encryption1RTT {
				now := time.Now()
				t.State.SetHandshakeCompletedTime(now)
			}
		},
	}
}

func NewStateTracer(state *State) *StateTracer {
	return &StateTracer{
		State: state,
	}
}
