package qlog_hquic

import "github.com/francoispqt/gojay"

type StateRequestReceivedEvent struct{}

func (t StateRequestReceivedEvent) Category() string                     { return "hquic" }
func (t StateRequestReceivedEvent) Name() string                         { return "state_request_received" }
func (t StateRequestReceivedEvent) IsNil() bool                          { return true }
func (t StateRequestReceivedEvent) MarshalJSONObject(enc *gojay.Encoder) {}

type StateParsedEvent struct{}

func (t StateParsedEvent) Category() string                     { return "hquic" }
func (t StateParsedEvent) Name() string                         { return "state_parsed" }
func (t StateParsedEvent) IsNil() bool                          { return true }
func (t StateParsedEvent) MarshalJSONObject(enc *gojay.Encoder) {}

type StateSerializedEvent struct{}

func (t StateSerializedEvent) Category() string                     { return "hquic" }
func (t StateSerializedEvent) Name() string                         { return "state_serialized" }
func (t StateSerializedEvent) IsNil() bool                          { return true }
func (t StateSerializedEvent) MarshalJSONObject(enc *gojay.Encoder) {}

type StateReceivedEvent struct{}

func (t StateReceivedEvent) Category() string                     { return "hquic" }
func (t StateReceivedEvent) Name() string                         { return "state_received" }
func (t StateReceivedEvent) IsNil() bool                          { return true }
func (t StateReceivedEvent) MarshalJSONObject(enc *gojay.Encoder) {}

type StateStoredEvent struct{}

func (t StateStoredEvent) Category() string                     { return "hquic" }
func (t StateStoredEvent) Name() string                         { return "state_stored" }
func (t StateStoredEvent) IsNil() bool                          { return true }
func (t StateStoredEvent) MarshalJSONObject(enc *gojay.Encoder) {}

type StateRestoredEvent struct{}

func (t StateRestoredEvent) Category() string                     { return "hquic" }
func (t StateRestoredEvent) Name() string                         { return "state_restored" }
func (t StateRestoredEvent) IsNil() bool                          { return true }
func (t StateRestoredEvent) MarshalJSONObject(enc *gojay.Encoder) {}

type StateSentEvent struct {
	PayloadBytes int
}

func (t StateSentEvent) Category() string { return "hquic" }
func (t StateSentEvent) Name() string     { return "state_sent" }
func (t StateSentEvent) IsNil() bool      { return false }
func (t StateSentEvent) MarshalJSONObject(enc *gojay.Encoder) {
	enc.IntKey("payload_bytes", t.PayloadBytes)
}

type UnknownConnectionPacketReceived struct{}

func (t UnknownConnectionPacketReceived) Category() string                     { return "hquic" }
func (t UnknownConnectionPacketReceived) Name() string                         { return "unknown_connection_packet_received" }
func (t UnknownConnectionPacketReceived) IsNil() bool                          { return true }
func (t UnknownConnectionPacketReceived) MarshalJSONObject(enc *gojay.Encoder) {}

type StateRequestSentEvent struct{}

func (t StateRequestSentEvent) Category() string                     { return "hquic" }
func (t StateRequestSentEvent) Name() string                         { return "state_request_sent" }
func (t StateRequestSentEvent) IsNil() bool                          { return true }
func (t StateRequestSentEvent) MarshalJSONObject(enc *gojay.Encoder) {}
