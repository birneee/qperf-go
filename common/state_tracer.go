package common

import (
	"context"
	"github.com/lucas-clemente/quic-go/logging"
	"net"
	"time"
)

type StateTracer struct {
	State *State
}

func (a StateTracer) TracerForConnection(ctx context.Context, p logging.Perspective, odcid logging.ConnectionID) logging.ConnectionTracer {
	return AckCountConnectionTracer{
		State: a.State,
	}
}

func (a StateTracer) SentPacket(addr net.Addr, header *logging.Header, count logging.ByteCount, frames []logging.Frame) {
	// ignore
}

func (a StateTracer) DroppedPacket(addr net.Addr, packetType logging.PacketType, count logging.ByteCount, reason logging.PacketDropReason) {
	// ignore
}

type AckCountConnectionTracer struct {
	State *State
}

func (a AckCountConnectionTracer) StartedConnection(local, remote net.Addr, srcConnID, destConnID logging.ConnectionID) {
	// ignore
}

func (a AckCountConnectionTracer) NegotiatedVersion(chosen logging.VersionNumber, clientVersions, serverVersions []logging.VersionNumber) {
	// ignore
}

func (a AckCountConnectionTracer) ClosedConnection(err error) {
	// ignore
}

func (a AckCountConnectionTracer) SentTransportParameters(parameters *logging.TransportParameters) {
	// ignore
}

func (a AckCountConnectionTracer) ReceivedTransportParameters(parameters *logging.TransportParameters) {
	// ignore
}

func (a AckCountConnectionTracer) RestoredTransportParameters(parameters *logging.TransportParameters) {
	// ignore
}

func (a AckCountConnectionTracer) SentPacket(hdr *logging.ExtendedHeader, size logging.ByteCount, ack *logging.AckFrame, frames []logging.Frame) {
	// ignore
}

func (a AckCountConnectionTracer) ReceivedVersionNegotiationPacket(header *logging.Header, numbers []logging.VersionNumber) {
	// ignore
}

func (a AckCountConnectionTracer) ReceivedRetry(header *logging.Header) {
	// ignore
}

func (a AckCountConnectionTracer) ReceivedPacket(hdr *logging.ExtendedHeader, size logging.ByteCount, frames []logging.Frame) {
	var receivedBytes uint64 = 0
	for _, frame := range frames {
		if streamFrame, ok := frame.(*logging.StreamFrame); ok {
			receivedBytes += uint64(streamFrame.Length)
		}
	}
	a.State.Add(receivedBytes, 1)
}

func (a AckCountConnectionTracer) BufferedPacket(packetType logging.PacketType) {
	// ignore
}

func (a AckCountConnectionTracer) DroppedPacket(packetType logging.PacketType, count logging.ByteCount, reason logging.PacketDropReason) {
	// ignore
}

func (a AckCountConnectionTracer) UpdatedMetrics(rttStats *logging.RTTStats, cwnd, bytesInFlight logging.ByteCount, packetsInFlight int) {
	// ignore
}

func (a AckCountConnectionTracer) AcknowledgedPacket(level logging.EncryptionLevel, number logging.PacketNumber) {
	// ignore
}

func (a AckCountConnectionTracer) LostPacket(level logging.EncryptionLevel, number logging.PacketNumber, reason logging.PacketLossReason) {
	// ignore
}

func (a AckCountConnectionTracer) UpdatedCongestionState(state logging.CongestionState) {
	// ignore
}

func (a AckCountConnectionTracer) UpdatedPTOCount(value uint32) {
	// ignore
}

func (a AckCountConnectionTracer) UpdatedKeyFromTLS(level logging.EncryptionLevel, perspective logging.Perspective) {
	// ignore
}

func (a AckCountConnectionTracer) UpdatedKey(generation logging.KeyPhase, remote bool) {
	// ignore
}

func (a AckCountConnectionTracer) DroppedEncryptionLevel(level logging.EncryptionLevel) {
	// ignore
}

func (a AckCountConnectionTracer) DroppedKey(generation logging.KeyPhase) {
	// ignore
}

func (a AckCountConnectionTracer) SetLossTimer(timerType logging.TimerType, level logging.EncryptionLevel, time time.Time) {
	// ignore
}

func (a AckCountConnectionTracer) LossTimerExpired(timerType logging.TimerType, level logging.EncryptionLevel) {
	// ignore
}

func (a AckCountConnectionTracer) LossTimerCanceled() {
	// ignore
}

func (a AckCountConnectionTracer) Close() {
	// ignore
}

func (a AckCountConnectionTracer) Debug(name, msg string) {
	// ignore
}
