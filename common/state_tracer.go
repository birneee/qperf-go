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
	return StateConnectionTracer{
		State: a.State,
	}
}

func (a StateTracer) SentPacket(addr net.Addr, header *logging.Header, count logging.ByteCount, frames []logging.Frame) {
	// ignore
}

func (a StateTracer) DroppedPacket(addr net.Addr, packetType logging.PacketType, count logging.ByteCount, reason logging.PacketDropReason) {
	// ignore
}

type StateConnectionTracer struct {
	State *State
}

func (a StateConnectionTracer) StartedConnection(local, remote net.Addr, srcConnID, destConnID logging.ConnectionID) {
	// ignore
}

func (a StateConnectionTracer) NegotiatedVersion(chosen logging.VersionNumber, clientVersions, serverVersions []logging.VersionNumber) {
	// ignore
}

func (a StateConnectionTracer) ClosedConnection(err error) {
	// ignore
}

func (a StateConnectionTracer) SentTransportParameters(parameters *logging.TransportParameters) {
	// ignore
}

func (a StateConnectionTracer) ReceivedTransportParameters(parameters *logging.TransportParameters) {
	// ignore
}

func (a StateConnectionTracer) RestoredTransportParameters(parameters *logging.TransportParameters) {
	// ignore
}

func (a StateConnectionTracer) SentPacket(hdr *logging.ExtendedHeader, size logging.ByteCount, ack *logging.AckFrame, frames []logging.Frame) {
	// ignore
}

func (a StateConnectionTracer) ReceivedVersionNegotiationPacket(header *logging.Header, numbers []logging.VersionNumber) {
	// ignore
}

func (a StateConnectionTracer) ReceivedRetry(header *logging.Header) {
	// ignore
}

func (a StateConnectionTracer) ReceivedPacket(hdr *logging.ExtendedHeader, size logging.ByteCount, frames []logging.Frame) {
	a.State.AddReceivedPackets(1)
}

func (a StateConnectionTracer) BufferedPacket(packetType logging.PacketType) {
	// ignore
}

func (a StateConnectionTracer) DroppedPacket(packetType logging.PacketType, count logging.ByteCount, reason logging.PacketDropReason) {
	// ignore
}

func (a StateConnectionTracer) UpdatedMetrics(rttStats *logging.RTTStats, cwnd, bytesInFlight logging.ByteCount, packetsInFlight int) {
	// ignore
}

func (a StateConnectionTracer) AcknowledgedPacket(level logging.EncryptionLevel, number logging.PacketNumber) {
	// ignore
}

func (a StateConnectionTracer) LostPacket(level logging.EncryptionLevel, number logging.PacketNumber, reason logging.PacketLossReason) {
	// ignore
}

func (a StateConnectionTracer) UpdatedCongestionState(state logging.CongestionState) {
	// ignore
}

func (a StateConnectionTracer) UpdatedPTOCount(value uint32) {
	// ignore
}

func (a StateConnectionTracer) UpdatedKeyFromTLS(level logging.EncryptionLevel, perspective logging.Perspective) {
	// ignore
}

func (a StateConnectionTracer) UpdatedKey(generation logging.KeyPhase, remote bool) {
	// ignore
}

func (a StateConnectionTracer) DroppedEncryptionLevel(level logging.EncryptionLevel) {
	// ignore
}

func (a StateConnectionTracer) DroppedKey(generation logging.KeyPhase) {
	// ignore
}

func (a StateConnectionTracer) SetLossTimer(timerType logging.TimerType, level logging.EncryptionLevel, time time.Time) {
	// ignore
}

func (a StateConnectionTracer) LossTimerExpired(timerType logging.TimerType, level logging.EncryptionLevel) {
	// ignore
}

func (a StateConnectionTracer) LossTimerCanceled() {
	// ignore
}

func (a StateConnectionTracer) Close() {
	// ignore
}

func (a StateConnectionTracer) Debug(name, msg string) {
	// ignore
}

func (a StateConnectionTracer) UpdatedPath(newRemote net.Addr) {
	// ignore
}
