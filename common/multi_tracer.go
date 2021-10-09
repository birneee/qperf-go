package common

import (
	"context"
	"github.com/lucas-clemente/quic-go/logging"
	"net"
	"time"
)

type MultiTracer struct {
	Tracers []logging.Tracer
}

func (m MultiTracer) TracerForConnection(ctx context.Context, p logging.Perspective, odcid logging.ConnectionID) logging.ConnectionTracer {
	connectionTracers := make([]logging.ConnectionTracer, len(m.Tracers))
	for index, tracer := range m.Tracers {
		connectionTracers[index] = tracer.TracerForConnection(ctx, p, odcid)
	}
	return MultiConnectionTracer{
		ConnectionTracers: connectionTracers,
	}
}

func (m MultiTracer) SentPacket(addr net.Addr, header *logging.Header, count logging.ByteCount, frames []logging.Frame) {
	for _, tracer := range m.Tracers {
		tracer.SentPacket(addr, header, count, frames)
	}
}

func (m MultiTracer) DroppedPacket(addr net.Addr, packetType logging.PacketType, count logging.ByteCount, reason logging.PacketDropReason) {
	for _, tracer := range m.Tracers {
		tracer.DroppedPacket(addr, packetType, count, reason)
	}
}

type MultiConnectionTracer struct {
	ConnectionTracers []logging.ConnectionTracer
}

func (m MultiConnectionTracer) StartedConnection(local, remote net.Addr, srcConnID, destConnID logging.ConnectionID) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.StartedConnection(local, remote, srcConnID, destConnID)
	}
}

func (m MultiConnectionTracer) NegotiatedVersion(chosen logging.VersionNumber, clientVersions, serverVersions []logging.VersionNumber) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.NegotiatedVersion(chosen, clientVersions, serverVersions)
	}
}

func (m MultiConnectionTracer) ClosedConnection(err error) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.ClosedConnection(err)
	}
}

func (m MultiConnectionTracer) SentTransportParameters(parameters *logging.TransportParameters) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.SentTransportParameters(parameters)
	}
}

func (m MultiConnectionTracer) ReceivedTransportParameters(parameters *logging.TransportParameters) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.ReceivedTransportParameters(parameters)
	}
}

func (m MultiConnectionTracer) RestoredTransportParameters(parameters *logging.TransportParameters) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.RestoredTransportParameters(parameters)
	}
}

func (m MultiConnectionTracer) SentPacket(hdr *logging.ExtendedHeader, size logging.ByteCount, ack *logging.AckFrame, frames []logging.Frame) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.SentPacket(hdr, size, ack, frames)
	}
}

func (m MultiConnectionTracer) ReceivedVersionNegotiationPacket(header *logging.Header, numbers []logging.VersionNumber) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.ReceivedVersionNegotiationPacket(header, numbers)
	}
}

func (m MultiConnectionTracer) ReceivedRetry(header *logging.Header) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.ReceivedRetry(header)
	}
}

func (m MultiConnectionTracer) ReceivedPacket(hdr *logging.ExtendedHeader, size logging.ByteCount, frames []logging.Frame) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.ReceivedPacket(hdr, size, frames)
	}
}

func (m MultiConnectionTracer) BufferedPacket(packetType logging.PacketType) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.BufferedPacket(packetType)
	}
}

func (m MultiConnectionTracer) DroppedPacket(packetType logging.PacketType, count logging.ByteCount, reason logging.PacketDropReason) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.DroppedPacket(packetType, count, reason)
	}
}

func (m MultiConnectionTracer) UpdatedMetrics(rttStats *logging.RTTStats, cwnd, bytesInFlight logging.ByteCount, packetsInFlight int) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.UpdatedMetrics(rttStats, cwnd, bytesInFlight, packetsInFlight)
	}
}

func (m MultiConnectionTracer) AcknowledgedPacket(level logging.EncryptionLevel, number logging.PacketNumber) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.AcknowledgedPacket(level, number)
	}
}

func (m MultiConnectionTracer) LostPacket(level logging.EncryptionLevel, number logging.PacketNumber, reason logging.PacketLossReason) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.LostPacket(level, number, reason)
	}
}

func (m MultiConnectionTracer) UpdatedCongestionState(state logging.CongestionState) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.UpdatedCongestionState(state)
	}
}

func (m MultiConnectionTracer) UpdatedPTOCount(value uint32) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.UpdatedPTOCount(value)
	}
}

func (m MultiConnectionTracer) UpdatedKeyFromTLS(level logging.EncryptionLevel, perspective logging.Perspective) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.UpdatedKeyFromTLS(level, perspective)
	}
}

func (m MultiConnectionTracer) UpdatedKey(generation logging.KeyPhase, remote bool) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.UpdatedKey(generation, remote)
	}
}

func (m MultiConnectionTracer) DroppedEncryptionLevel(level logging.EncryptionLevel) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.DroppedEncryptionLevel(level)
	}
}

func (m MultiConnectionTracer) DroppedKey(generation logging.KeyPhase) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.DroppedKey(generation)
	}
}

func (m MultiConnectionTracer) SetLossTimer(timerType logging.TimerType, level logging.EncryptionLevel, time time.Time) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.SetLossTimer(timerType, level, time)
	}
}

func (m MultiConnectionTracer) LossTimerExpired(timerType logging.TimerType, level logging.EncryptionLevel) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.LossTimerExpired(timerType, level)
	}
}

func (m MultiConnectionTracer) LossTimerCanceled() {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.LossTimerCanceled()
	}
}

func (m MultiConnectionTracer) Close() {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.Close()
	}
}

func (m MultiConnectionTracer) Debug(name, msg string) {
	for _, connectionTracer := range m.ConnectionTracers {
		connectionTracer.Debug(name, msg)
	}
}
