package common

import (
	"context"
	"github.com/lucas-clemente/quic-go/logging"
	"net"
	"time"
)

type migrationTracer struct {
	onMigration func(addr net.Addr)
}

func NewMigrationTracer(onMigration func(addr net.Addr)) *migrationTracer {
	return &migrationTracer{
		onMigration: onMigration,
	}
}

func (a migrationTracer) TracerForConnection(ctx context.Context, p logging.Perspective, odcid logging.ConnectionID) logging.ConnectionTracer {
	return connectionTracer{
		onMigration: a.onMigration,
	}
}

func (a migrationTracer) SentPacket(addr net.Addr, header *logging.Header, count logging.ByteCount, frames []logging.Frame) {
	// ignore
}

func (a migrationTracer) DroppedPacket(addr net.Addr, packetType logging.PacketType, count logging.ByteCount, reason logging.PacketDropReason) {
	// ignore
}

type connectionTracer struct {
	onMigration func(addr net.Addr)
}

func (a connectionTracer) StartedConnection(local, remote net.Addr, srcConnID, destConnID logging.ConnectionID) {
	// ignore
}

func (a connectionTracer) NegotiatedVersion(chosen logging.VersionNumber, clientVersions, serverVersions []logging.VersionNumber) {
	// ignore
}

func (a connectionTracer) ClosedConnection(err error) {
	// ignore
}

func (a connectionTracer) SentTransportParameters(parameters *logging.TransportParameters) {
	// ignore
}

func (a connectionTracer) ReceivedTransportParameters(parameters *logging.TransportParameters) {
	// ignore
}

func (a connectionTracer) RestoredTransportParameters(parameters *logging.TransportParameters) {
	// ignore
}

func (a connectionTracer) SentPacket(hdr *logging.ExtendedHeader, size logging.ByteCount, ack *logging.AckFrame, frames []logging.Frame) {
	// ignore
}

func (a connectionTracer) ReceivedVersionNegotiationPacket(header *logging.Header, numbers []logging.VersionNumber) {
	// ignore
}

func (a connectionTracer) ReceivedRetry(header *logging.Header) {
	// ignore
}

func (a connectionTracer) ReceivedPacket(hdr *logging.ExtendedHeader, size logging.ByteCount, frames []logging.Frame) {
	// ignore
}

func (a connectionTracer) BufferedPacket(packetType logging.PacketType) {
	// ignore
}

func (a connectionTracer) DroppedPacket(packetType logging.PacketType, count logging.ByteCount, reason logging.PacketDropReason) {
	// ignore
}

func (a connectionTracer) UpdatedMetrics(rttStats *logging.RTTStats, cwnd, bytesInFlight logging.ByteCount, packetsInFlight int) {
	// ignore
}

func (a connectionTracer) AcknowledgedPacket(level logging.EncryptionLevel, number logging.PacketNumber) {
	// ignore
}

func (a connectionTracer) LostPacket(level logging.EncryptionLevel, number logging.PacketNumber, reason logging.PacketLossReason) {
	// ignore
}

func (a connectionTracer) UpdatedCongestionState(state logging.CongestionState) {
	// ignore
}

func (a connectionTracer) UpdatedPTOCount(value uint32) {
	// ignore
}

func (a connectionTracer) UpdatedKeyFromTLS(level logging.EncryptionLevel, perspective logging.Perspective) {
	// ignore
}

func (a connectionTracer) UpdatedKey(generation logging.KeyPhase, remote bool) {
	// ignore
}

func (a connectionTracer) DroppedEncryptionLevel(level logging.EncryptionLevel) {
	// ignore
}

func (a connectionTracer) DroppedKey(generation logging.KeyPhase) {
	// ignore
}

func (a connectionTracer) SetLossTimer(timerType logging.TimerType, level logging.EncryptionLevel, time time.Time) {
	// ignore
}

func (a connectionTracer) LossTimerExpired(timerType logging.TimerType, level logging.EncryptionLevel) {
	// ignore
}

func (a connectionTracer) LossTimerCanceled() {
	// ignore
}

func (a connectionTracer) Close() {
	// ignore
}

func (a connectionTracer) Debug(name, msg string) {
	// ignore
}

func (a connectionTracer) UpdatedPath(newRemote net.Addr) {
	a.onMigration(newRemote)
}
