package qlog_quic

// mostly copied from quic-go/qlog/qlog.go

import (
	"context"
	"github.com/francoispqt/gojay"
	"github.com/quic-go/quic-go/logging"
	"net"
	"qperf-go/common/qlog"
	"time"
)

type QlogWriterConnectionTracer interface {
	logging.ConnectionTracer
	QlogWriter() qlog.QlogWriter
}

type connectionTracer struct {
	qlogWriter                        qlog.QlogWriter
	odcid                             string
	perspective                       logging.Perspective
	lastMetrics                       *metrics
	groupID                           string
	fastPathIncludePacketReceived     bool
	fastPathIncludePacketSent         bool
	fastPathIncludeMetricsUpdated     bool
	fastPathIncludeLossTimerSet       bool
	fastPathIncludeXadsRecordReceived bool
}

var _ logging.ConnectionTracer = &connectionTracer{}
var _ QlogWriterConnectionTracer = &connectionTracer{}

// NewTracer creates a new tracer to record a qlog for a connection.
func NewTracer(qlogWriter qlog.QlogWriter) func(ctx context.Context, p logging.Perspective, id logging.ConnectionID) logging.ConnectionTracer {
	return func(ctx context.Context, p logging.Perspective, id logging.ConnectionID) logging.ConnectionTracer {
		return NewConnectionTracer(qlogWriter, p, id)
	}
}

// NewConnectionTracer creates a new tracer to record a qlog for a connection.
func NewConnectionTracer(qlogWriter qlog.QlogWriter, p logging.Perspective, odcid logging.ConnectionID) logging.ConnectionTracer {
	t := &connectionTracer{
		qlogWriter:  qlogWriter,
		perspective: p,
		odcid:       odcid.String(),
		groupID:     odcid.String(),
		//TODO add more fast paths for performance critical events
		fastPathIncludePacketReceived:     qlogWriter.Includes(eventPacketReceived{}.Category(), eventPacketReceived{}.Name()),
		fastPathIncludePacketSent:         qlogWriter.Includes(eventPacketSent{}.Category(), eventPacketSent{}.Name()),
		fastPathIncludeMetricsUpdated:     qlogWriter.Includes(eventMetricsUpdated{}.Category(), eventMetricsUpdated{}.Name()),
		fastPathIncludeLossTimerSet:       qlogWriter.Includes(eventLossTimerSet{}.Category(), eventLossTimerSet{}.Name()),
		fastPathIncludeXadsRecordReceived: qlogWriter.Includes(eventXadsRecordReceived{}.Category(), eventXadsRecordReceived{}.Name()),
	}
	return t
}

func (t *connectionTracer) Close() {
	// do nothing
}

func (t *connectionTracer) recordEvent(eventTime time.Time, details qlog.EventDetails) {
	t.qlogWriter.RecordEventWithTimeGroupODCID(details, eventTime, t.groupID, t.odcid)
}

func (t *connectionTracer) QlogWriter() qlog.QlogWriter {
	return t.qlogWriter
}

func (t *connectionTracer) StartedConnection(local, remote net.Addr, srcConnID, destConnID logging.ConnectionID) {
	// ignore this event if we're not dealing with UDP addresses here
	localAddr, ok := local.(*net.UDPAddr)
	if !ok {
		return
	}
	remoteAddr, ok := remote.(*net.UDPAddr)
	if !ok {
		return
	}
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventConnectionStarted{
		SrcAddr:          localAddr,
		DestAddr:         remoteAddr,
		SrcConnectionID:  srcConnID,
		DestConnectionID: destConnID,
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) NegotiatedVersion(chosen logging.VersionNumber, client, server []logging.VersionNumber) {
	var clientVersions, serverVersions []versionNumber
	if len(client) > 0 {
		clientVersions = make([]versionNumber, len(client))
		for i, v := range client {
			clientVersions[i] = versionNumber(v)
		}
	}
	if len(server) > 0 {
		serverVersions = make([]versionNumber, len(server))
		for i, v := range server {
			serverVersions[i] = versionNumber(v)
		}
	}
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventVersionNegotiated{
		clientVersions: clientVersions,
		serverVersions: serverVersions,
		chosenVersion:  versionNumber(chosen),
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) ClosedConnection(e error) {
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventConnectionClosed{e: e})
	//t.mutex.Unlock()
}

func (t *connectionTracer) SentTransportParameters(tp *logging.TransportParameters) {
	t.recordTransportParameters(t.perspective, tp)
}

func (t *connectionTracer) ReceivedTransportParameters(tp *logging.TransportParameters) {
	t.recordTransportParameters(t.perspective.Opposite(), tp)
}

func (t *connectionTracer) RestoredTransportParameters(tp *logging.TransportParameters) {
	ev := t.toTransportParameters(tp)
	ev.Restore = true

	//t.mutex.Lock()
	t.recordEvent(time.Now(), ev)
	//t.mutex.Unlock()
}

func (t *connectionTracer) recordTransportParameters(sentBy logging.Perspective, tp *logging.TransportParameters) {
	ev := t.toTransportParameters(tp)
	ev.Owner = ownerLocal
	if sentBy != t.perspective {
		ev.Owner = ownerRemote
	}
	ev.SentBy = sentBy

	//t.mutex.Lock()
	t.recordEvent(time.Now(), ev)
	//t.mutex.Unlock()
}

func (t *connectionTracer) toTransportParameters(tp *logging.TransportParameters) *eventTransportParameters {
	var pa *preferredAddress
	if tp.PreferredAddress != nil {
		pa = &preferredAddress{
			IPv4:                tp.PreferredAddress.IPv4,
			PortV4:              tp.PreferredAddress.IPv4Port,
			IPv6:                tp.PreferredAddress.IPv6,
			PortV6:              tp.PreferredAddress.IPv6Port,
			ConnectionID:        tp.PreferredAddress.ConnectionID,
			StatelessResetToken: tp.PreferredAddress.StatelessResetToken,
		}
	}
	return &eventTransportParameters{
		OriginalDestinationConnectionID: tp.OriginalDestinationConnectionID,
		InitialSourceConnectionID:       tp.InitialSourceConnectionID,
		RetrySourceConnectionID:         tp.RetrySourceConnectionID,
		StatelessResetToken:             tp.StatelessResetToken,
		DisableActiveMigration:          tp.DisableActiveMigration,
		MaxIdleTimeout:                  tp.MaxIdleTimeout,
		MaxUDPPayloadSize:               tp.MaxUDPPayloadSize,
		AckDelayExponent:                tp.AckDelayExponent,
		MaxAckDelay:                     tp.MaxAckDelay,
		ActiveConnectionIDLimit:         tp.ActiveConnectionIDLimit,
		InitialMaxData:                  tp.InitialMaxData,
		InitialMaxStreamDataBidiLocal:   tp.InitialMaxStreamDataBidiLocal,
		InitialMaxStreamDataBidiRemote:  tp.InitialMaxStreamDataBidiRemote,
		InitialMaxStreamDataUni:         tp.InitialMaxStreamDataUni,
		InitialMaxStreamsBidi:           int64(tp.MaxBidiStreamNum),
		InitialMaxStreamsUni:            int64(tp.MaxUniStreamNum),
		PreferredAddress:                pa,
		MaxDatagramFrameSize:            tp.MaxDatagramFrameSize,
	}
}

func (t *connectionTracer) SentLongHeaderPacket(hdr *logging.ExtendedHeader, packetSize logging.ByteCount, ack *logging.AckFrame, frames []logging.Frame) {
	t.sentPacket(*transformLongHeader(hdr), packetSize, hdr.Length, ack, frames)
}

func (t *connectionTracer) SentShortHeaderPacket(hdr *logging.ShortHeader, packetSize logging.ByteCount, ack *logging.AckFrame, frames []logging.Frame) {
	if !t.fastPathIncludePacketSent {
		return
	}
	t.sentPacket(*transformShortHeader(hdr), packetSize, 0, ack, frames)
}

func (t *connectionTracer) sentPacket(hdr gojay.MarshalerJSONObject, packetSize, payloadLen logging.ByteCount, ack *logging.AckFrame, frames []logging.Frame) {
	numFrames := len(frames)
	if ack != nil {
		numFrames++
	}
	fs := make([]frame, 0, numFrames)
	if ack != nil {
		fs = append(fs, frame{Frame: ack})
	}
	for _, f := range frames {
		fs = append(fs, frame{Frame: f})
	}
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventPacketSent{
		Header:        hdr,
		Length:        packetSize,
		PayloadLength: payloadLen,
		Frames:        fs,
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) ReceivedLongHeaderPacket(hdr *logging.ExtendedHeader, packetSize logging.ByteCount, frames []logging.Frame) {
	fs := make([]frame, len(frames))
	for i, f := range frames {
		fs[i] = frame{Frame: f}
	}
	header := *transformLongHeader(hdr)
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventPacketReceived{
		Header:        header,
		Length:        packetSize,
		PayloadLength: hdr.Length,
		Frames:        fs,
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) ReceivedShortHeaderPacket(hdr *logging.ShortHeader, packetSize logging.ByteCount, frames []logging.Frame) {
	if !t.fastPathIncludePacketReceived {
		return
	}
	fs := make([]frame, len(frames))
	for i, f := range frames {
		fs[i] = frame{Frame: f}
	}
	header := *transformShortHeader(hdr)
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventPacketReceived{
		Header:        header,
		Length:        packetSize,
		PayloadLength: packetSize - ShortHeaderLen(hdr.DestConnectionID, uint8(hdr.PacketNumberLen)),
		Frames:        fs,
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) ReceivedRetry(hdr *logging.Header) {
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventRetryReceived{
		Header: *transformHeader(hdr),
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) ReceivedVersionNegotiationPacket(dest, src logging.ArbitraryLenConnectionID, versions []logging.VersionNumber) {
	ver := make([]versionNumber, len(versions))
	for i, v := range versions {
		ver[i] = versionNumber(v)
	}
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventVersionNegotiationReceived{
		Header: packetHeaderVersionNegotiation{
			SrcConnectionID:  src,
			DestConnectionID: dest,
		},
		SupportedVersions: ver,
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) BufferedPacket(pt logging.PacketType, size logging.ByteCount) {
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventPacketBuffered{
		PacketType: pt,
		PacketSize: size,
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) DroppedPacket(pt logging.PacketType, size logging.ByteCount, reason logging.PacketDropReason) {
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventPacketDropped{
		PacketType: pt,
		PacketSize: size,
		Trigger:    packetDropReason(reason),
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) UpdatedMetrics(rttStats *logging.RTTStats, cwnd, bytesInFlight logging.ByteCount, packetsInFlight int) {
	if !t.fastPathIncludeMetricsUpdated {
		return
	}
	m := &metrics{
		MinRTT:           rttStats.MinRTT(),
		SmoothedRTT:      rttStats.SmoothedRTT(),
		LatestRTT:        rttStats.LatestRTT(),
		RTTVariance:      rttStats.MeanDeviation(),
		CongestionWindow: cwnd,
		BytesInFlight:    bytesInFlight,
		PacketsInFlight:  packetsInFlight,
	}
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventMetricsUpdated{
		Last:    t.lastMetrics,
		Current: m,
	})
	t.lastMetrics = m
	//t.mutex.Unlock()
}

func (t *connectionTracer) AcknowledgedPacket(logging.EncryptionLevel, logging.PacketNumber) {}

func (t *connectionTracer) LostPacket(encLevel logging.EncryptionLevel, pn logging.PacketNumber, lossReason logging.PacketLossReason) {
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventPacketLost{
		PacketType:   getPacketTypeFromEncryptionLevel(encLevel),
		PacketNumber: pn,
		Trigger:      packetLossReason(lossReason),
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) UpdatedCongestionState(state logging.CongestionState) {
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventCongestionStateUpdated{state: congestionState(state)})
	//t.mutex.Unlock()
}

func (t *connectionTracer) UpdatedPTOCount(value uint32) {
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventUpdatedPTO{Value: value})
	//t.mutex.Unlock()
}

func (t *connectionTracer) UpdatedKeyFromTLS(encLevel logging.EncryptionLevel, pers logging.Perspective) {
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventKeyUpdated{
		Trigger: keyUpdateTLS,
		KeyType: encLevelToKeyType(encLevel, pers),
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) UpdatedKey(generation logging.KeyPhase, remote bool) {
	trigger := keyUpdateLocal
	if remote {
		trigger = keyUpdateRemote
	}
	//t.mutex.Lock()
	now := time.Now()
	t.recordEvent(now, &eventKeyUpdated{
		Trigger:    trigger,
		KeyType:    keyTypeClient1RTT,
		Generation: generation,
	})
	t.recordEvent(now, &eventKeyUpdated{
		Trigger:    trigger,
		KeyType:    keyTypeServer1RTT,
		Generation: generation,
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) DroppedEncryptionLevel(encLevel logging.EncryptionLevel) {
	//t.mutex.Lock()
	now := time.Now()
	if encLevel == logging.Encryption0RTT {
		t.recordEvent(now, &eventKeyDiscarded{KeyType: encLevelToKeyType(encLevel, t.perspective)})
	} else {
		t.recordEvent(now, &eventKeyDiscarded{KeyType: encLevelToKeyType(encLevel, logging.PerspectiveServer)})
		t.recordEvent(now, &eventKeyDiscarded{KeyType: encLevelToKeyType(encLevel, logging.PerspectiveClient)})
	}
	//t.mutex.Unlock()
}

func (t *connectionTracer) DroppedKey(generation logging.KeyPhase) {
	//t.mutex.Lock()
	now := time.Now()
	t.recordEvent(now, &eventKeyDiscarded{
		KeyType:    encLevelToKeyType(logging.Encryption1RTT, logging.PerspectiveServer),
		Generation: generation,
	})
	t.recordEvent(now, &eventKeyDiscarded{
		KeyType:    encLevelToKeyType(logging.Encryption1RTT, logging.PerspectiveClient),
		Generation: generation,
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) SetLossTimer(tt logging.TimerType, encLevel logging.EncryptionLevel, timeout time.Time) {
	if !t.fastPathIncludeLossTimerSet {
		return
	}
	//t.mutex.Lock()
	now := time.Now()
	t.recordEvent(now, &eventLossTimerSet{
		TimerType: timerType(tt),
		EncLevel:  encLevel,
		Delta:     timeout.Sub(now),
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) LossTimerExpired(tt logging.TimerType, encLevel logging.EncryptionLevel) {
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventLossTimerExpired{
		TimerType: timerType(tt),
		EncLevel:  encLevel,
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) LossTimerCanceled() {
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventLossTimerCanceled{})
	//t.mutex.Unlock()
}

func (t *connectionTracer) Debug(name, msg string) {
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventGeneric{
		name: name,
		msg:  msg,
	})
	//t.mutex.Unlock()
}

func (t *connectionTracer) XadsReceiveRecord(streamID logging.StreamID, rawLength int, dataLength int) {
	//TODO this event is not standardized by https://datatracker.ietf.org/doc/html/draft-marx-qlog-event-definitions-quic-h3
	if !t.fastPathIncludeXadsRecordReceived {
		return
	}
	//t.mutex.Lock()
	t.recordEvent(time.Now(), &eventXadsRecordReceived{
		streamID:   streamID,
		rawLength:  rawLength,
		dataLength: dataLength,
	})
	//t.mutex.Unlock()
}
