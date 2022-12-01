package common

import (
	"context"
	"github.com/lucas-clemente/quic-go/logging"
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

func (n StateConnectionTracer) ReceivedLongHeaderPacket(*logging.ExtendedHeader, logging.ByteCount, []logging.Frame) {
	n.State.AddReceivedPackets(1)
}

func (n StateConnectionTracer) ReceivedShortHeaderPacket(*logging.ShortHeader, logging.ByteCount, []logging.Frame) {
	n.State.AddReceivedPackets(1)

}
