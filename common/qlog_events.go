package common

import (
	"github.com/francoispqt/gojay"
	"github.com/quic-go/quic-go/logging"
	"qperf-go/common/qlog"
	"time"
)

type TestEvent struct{}

var _ qlog.EventDetails = &TestEvent{}

func (t TestEvent) Category() string { return "qperf" }
func (t TestEvent) Name() string     { return "test" }
func (t TestEvent) IsNil() bool      { return false }
func (t TestEvent) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey("key", "value")
}

type HandshakeCompletedEvent struct{}

var _ qlog.EventDetails = &HandshakeCompletedEvent{}

func (t HandshakeCompletedEvent) Category() string                     { return "qperf" }
func (t HandshakeCompletedEvent) Name() string                         { return "handshake_completed" }
func (t HandshakeCompletedEvent) IsNil() bool                          { return true }
func (t HandshakeCompletedEvent) MarshalJSONObject(enc *gojay.Encoder) {}

type HandshakeConfirmedEvent struct{}

var _ qlog.EventDetails = &HandshakeConfirmedEvent{}

func (t HandshakeConfirmedEvent) Category() string                     { return "qperf" }
func (t HandshakeConfirmedEvent) Name() string                         { return "handshake_confirmed" }
func (t HandshakeConfirmedEvent) IsNil() bool                          { return true }
func (t HandshakeConfirmedEvent) MarshalJSONObject(enc *gojay.Encoder) {}

type FirstAppDataReceivedEvent struct{}

var _ qlog.EventDetails = &FirstAppDataReceivedEvent{}

func (t FirstAppDataReceivedEvent) Category() string                     { return "qperf" }
func (t FirstAppDataReceivedEvent) Name() string                         { return "first_app_data_received" }
func (t FirstAppDataReceivedEvent) IsNil() bool                          { return true }
func (t FirstAppDataReceivedEvent) MarshalJSONObject(enc *gojay.Encoder) {}

type ReportEvent struct {
	Period                            time.Duration
	StreamMegaBitsPerSecondReceived   *float32
	StreamBytesReceived               *logging.ByteCount
	PacketsReceived                   *uint64
	MinRTT                            *time.Duration
	MaxRTT                            *time.Duration
	PacketsLost                       *uint64
	StreamBytesSent                   *logging.ByteCount
	DatagramBytesReceived             *logging.ByteCount
	DatagramBytesSent                 *logging.ByteCount
	DatagramMegaBitsPerSecondReceived *float32
	DatagramMegaBitsPerSecondSent     *float32
	StreamMegaBitsPerSecondSent       *float32
}

var _ qlog.EventDetails = &ReportEvent{}

func (t ReportEvent) Category() string { return "qperf" }
func (t ReportEvent) Name() string     { return "report" }
func (t ReportEvent) IsNil() bool      { return false }
func (t ReportEvent) MarshalJSONObject(enc *gojay.Encoder) {
	if t.StreamMegaBitsPerSecondReceived != nil {
		enc.Float32Key("stream_mbps_received", *t.StreamMegaBitsPerSecondReceived)
	}
	if t.StreamMegaBitsPerSecondSent != nil {
		enc.Float32Key("stream_mbps_sent", *t.StreamMegaBitsPerSecondSent)
	}
	if t.StreamBytesReceived != nil {
		enc.Uint64Key("stream_bytes_received", uint64(*t.StreamBytesReceived))
	}
	if t.StreamBytesSent != nil {
		enc.Uint64Key("stream_bytes_sent", uint64(*t.StreamBytesSent))
	}
	if t.DatagramMegaBitsPerSecondReceived != nil {
		enc.Float32Key("datagram_mbps_received", *t.DatagramMegaBitsPerSecondReceived)
	}
	if t.DatagramMegaBitsPerSecondSent != nil {
		enc.Float32Key("datagram_mbps_sent", *t.DatagramMegaBitsPerSecondSent)
	}
	if t.DatagramBytesSent != nil {
		enc.Uint64Key("datagram_bytes_sent", uint64(*t.DatagramBytesSent))
	}
	if t.DatagramBytesReceived != nil {
		enc.Uint64Key("datagram_bytes_received", uint64(*t.DatagramBytesReceived))
	}
	if t.PacketsReceived != nil {
		enc.Uint64Key("packets_received", *t.PacketsReceived)
	}
	if t.MinRTT != nil {
		enc.Float32Key("min_rtt", float32(t.MinRTT.Seconds()*1000))
	}
	if t.MaxRTT != nil {
		enc.Float32Key("max_rtt", float32(t.MaxRTT.Seconds()*1000))
	}
	if t.PacketsLost != nil {
		enc.Uint64Key("packets_lost", *t.PacketsLost)
	}
	enc.Float32Key("period", float32(t.Period.Seconds()*1000))
}

type TotalEvent struct {
	ReportEvent
}

var _ qlog.EventDetails = &TotalEvent{}

func (t TotalEvent) Name() string { return "total" }
