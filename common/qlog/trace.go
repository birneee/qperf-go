package qlog

// mostly copied from quic-go/qlog/trace.go

import (
	"github.com/quic-go/quic-go/logging"
	"time"

	"github.com/francoispqt/gojay"
)

type topLevel struct {
	trace trace
}

func (l *topLevel) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	switch key {
	case "trace":
		return dec.Object(&l.trace)
	}
	return nil
}

func (l *topLevel) NKeys() int {
	return 0 // parse all keys
}

func (topLevel) IsNil() bool { return false }
func (l topLevel) MarshalJSONObject(enc *gojay.Encoder) {
	//TODO replace with JSON-SEQ as defined by qlog draft-04, as soon as qvis supports it
	enc.StringKey("qlog_format", "NDJSON")
	//TODO replace with 0.4 as defined by qlog draft-04, as soon as qvis supports it
	enc.StringKey("qlog_version", "draft-02")
	enc.StringKeyOmitEmpty("title", l.trace.Title)
	enc.StringKeyOmitEmpty("code_version", l.trace.CodeVersion)
	enc.ObjectKey("trace", l.trace)
}

type vantagePoint struct {
	Name string
	Type logging.Perspective
}

func (p vantagePoint) IsNil() bool { return len(p.Name) == 0 && p.Type == 0 }
func (p vantagePoint) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty("name", p.Name)
	switch p.Type {
	case logging.PerspectiveClient:
		enc.StringKey("type", "client")
	case logging.PerspectiveServer:
		enc.StringKey("type", "server")
	}
}

type commonFields struct {
	ODCID         string
	GroupID       string
	ProtocolType  string
	ReferenceTime time.Time
}

func (f *commonFields) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	switch key {
	case "ODCID":
		return dec.String(&f.ODCID)
	case "group_id":
		return dec.String(&f.GroupID)
	}
	return nil
}

func (f *commonFields) NKeys() int {
	return 0 // parse all keys
}

func (f commonFields) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty("ODCID", f.ODCID)
	enc.StringKeyOmitEmpty("group_id", f.GroupID)
	enc.StringKeyOmitEmpty("protocol_type", f.ProtocolType)
	enc.Float64Key("reference_time", float64(f.ReferenceTime.UnixNano())/1e6)
	enc.StringKey("time_format", "relative")
}

func (f commonFields) IsNil() bool { return false }

type trace struct {
	Title        string
	CodeVersion  string
	VantagePoint vantagePoint
	CommonFields commonFields
}

func (t *trace) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	switch key {
	case "common_fields":
		return dec.Object(&t.CommonFields)
	}
	return nil
}

func (t *trace) NKeys() int {
	return 0 // parse all keys
}

func (trace) IsNil() bool { return false }
func (t trace) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKeyOmitEmpty("vantage_point", t.VantagePoint)
	enc.ObjectKey("common_fields", t.CommonFields)
}
