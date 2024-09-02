package common

import (
	"errors"
	"fmt"
	"github.com/francoispqt/gojay"
	"github.com/quic-go/quic-go"
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

func (t HandshakeCompletedEvent) Category() string                   { return "qperf" }
func (t HandshakeCompletedEvent) Name() string                       { return "handshake_completed" }
func (t HandshakeCompletedEvent) IsNil() bool                        { return true }
func (t HandshakeCompletedEvent) MarshalJSONObject(_ *gojay.Encoder) {}

type HandshakeConfirmedEvent struct{}

var _ qlog.EventDetails = &HandshakeConfirmedEvent{}

func (t HandshakeConfirmedEvent) Category() string                   { return "qperf" }
func (t HandshakeConfirmedEvent) Name() string                       { return "handshake_confirmed" }
func (t HandshakeConfirmedEvent) IsNil() bool                        { return true }
func (t HandshakeConfirmedEvent) MarshalJSONObject(_ *gojay.Encoder) {}

type FirstAppDataReceivedEvent struct{}

var _ qlog.EventDetails = &FirstAppDataReceivedEvent{}

func (t FirstAppDataReceivedEvent) Category() string                   { return "qperf" }
func (t FirstAppDataReceivedEvent) Name() string                       { return "first_app_data_received" }
func (t FirstAppDataReceivedEvent) IsNil() bool                        { return true }
func (t FirstAppDataReceivedEvent) MarshalJSONObject(_ *gojay.Encoder) {}

type FirstAppDataSentEvent struct{}

var _ qlog.EventDetails = &FirstAppDataSentEvent{}

func (t FirstAppDataSentEvent) Category() string                   { return "qperf" }
func (t FirstAppDataSentEvent) Name() string                       { return "first_app_data_sent" }
func (t FirstAppDataSentEvent) IsNil() bool                        { return true }
func (t FirstAppDataSentEvent) MarshalJSONObject(_ *gojay.Encoder) {}

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
	DeadlineExceededResponses         *uint64
	ResponsesReceived                 *uint64
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
	if t.ResponsesReceived != nil {
		enc.Uint64KeyOmitEmpty("responses_received", *t.ResponsesReceived)
	}
	if t.DeadlineExceededResponses != nil {
		enc.Uint64Key("deadline_exceeded", *t.DeadlineExceededResponses)
	}
	enc.Float32Key("period", float32(t.Period.Seconds()*1000))
}

type TotalEvent struct {
	ReportEvent
}

var _ qlog.EventDetails = &TotalEvent{}

func (t TotalEvent) Name() string { return "total" }

type EventConnectionStarted struct {
	DestConnectionID logging.ConnectionID
}

var _ qlog.EventDetails = &EventConnectionStarted{}

func (e EventConnectionStarted) Category() string { return "transport" }
func (e EventConnectionStarted) Name() string     { return "connection_started" }
func (e EventConnectionStarted) IsNil() bool      { return false }

func (e EventConnectionStarted) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey("dst_cid", e.DestConnectionID.String())
}

type EventConnectionClosed struct {
	Err error
}

func (e EventConnectionClosed) Category() string { return "transport" }
func (e EventConnectionClosed) Name() string     { return "connection_closed" }
func (e EventConnectionClosed) IsNil() bool      { return false }

func (e EventConnectionClosed) MarshalJSONObject(enc *gojay.Encoder) {
	var (
		statelessResetErr     *quic.StatelessResetError
		handshakeTimeoutErr   *quic.HandshakeTimeoutError
		idleTimeoutErr        *quic.IdleTimeoutError
		applicationErr        *quic.ApplicationError
		transportErr          *quic.TransportError
		versionNegotiationErr *quic.VersionNegotiationError
	)
	switch {
	case errors.As(e.Err, &statelessResetErr):
		enc.StringKey("owner", "remote")
		enc.StringKey("trigger", "stateless_reset")
		enc.StringKey("stateless_reset_token", fmt.Sprintf("%x", statelessResetErr.Token))
	case errors.As(e.Err, &handshakeTimeoutErr):
		enc.StringKey("owner", "local")
		enc.StringKey("trigger", "handshake_timeout")
	case errors.As(e.Err, &idleTimeoutErr):
		enc.StringKey("owner", "local")
		enc.StringKey("trigger", "idle_timeout")
	case errors.As(e.Err, &applicationErr):
		owner := "local"
		if applicationErr.Remote {
			owner = "remote"
		}
		enc.StringKey("owner", owner)
		enc.Uint64Key("application_code", uint64(applicationErr.ErrorCode))
		enc.StringKey("reason", applicationErr.ErrorMessage)
	case errors.As(e.Err, &transportErr):
		owner := "local"
		if transportErr.Remote {
			owner = "remote"
		}
		enc.StringKey("owner", owner)
		enc.StringKey("connection_code", transportErr.ErrorCode.String())
		enc.StringKey("reason", transportErr.ErrorMessage)
	case errors.As(e.Err, &versionNegotiationErr):
		enc.StringKey("trigger", "version_mismatch")
	}
}

type EventGeneric struct {
	CategoryF string
	NameF     string
	MsgF      string
}

func (e EventGeneric) Category() string { return e.CategoryF }
func (e EventGeneric) Name() string     { return e.NameF }
func (e EventGeneric) IsNil() bool      { return false }

func (e EventGeneric) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey("details", e.MsgF)
}
