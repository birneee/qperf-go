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

func (a StateConnectionTracer) ReceivedPacket(hdr *logging.ExtendedHeader, size logging.ByteCount, frames []logging.Frame) {
	a.State.AddReceivedPackets(1)
}
