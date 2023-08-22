package qlog

// inspired quic-go/qlog/event.go

import (
	"github.com/francoispqt/gojay"
	"strings"
	"time"
)

func milliseconds(dur time.Duration) float64 { return float64(dur.Nanoseconds()) / 1e6 }

func float2Milliseconds(f float64) time.Duration {
	return time.Duration(f * 1e6)
}

type EventDetails interface {
	Category() string
	Name() string
	gojay.MarshalerJSONObject
}

type event struct {
	RelativeTime time.Duration
	EventDetails
	GroupID string
	ODCID   string
}

var _ gojay.MarshalerJSONObject = &event{}
var _ gojay.UnmarshalerJSONObject = &event{}

func (e event) IsNil() bool { return false }

// MarshalJSONObject implements gojay.MarshalJSONObject
func (e event) MarshalJSONObject(enc *gojay.Encoder) {
	enc.Float64Key("time", milliseconds(e.RelativeTime))
	enc.StringKey("name", e.Category()+":"+e.Name())
	enc.ObjectKey("data", e.EventDetails)
	enc.StringKeyOmitEmpty("group_id", e.GroupID)
	enc.StringKeyOmitEmpty("ODCID", e.ODCID)
}

// UnmarshalJSONObject implements gojay.UnmarshalerJSONObject
func (e event) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	if e.EventDetails == nil {
		e.EventDetails = &genericEventDetails{}
	}
	switch key {
	case "time":
		var ms float64
		err := dec.Float64(&ms)
		if err != nil {
			return err
		}
		e.RelativeTime = float2Milliseconds(ms)
	case "name":
		ged := e.EventDetails.(*genericEventDetails)
		var fullName string
		err := dec.String(&fullName)
		if err != nil {
			return err
		}
		parts := strings.Split(fullName, ":")
		ged.category = parts[0]
		ged.name = parts[1]
	case "data":
		ged := e.EventDetails.(*genericEventDetails)
		err := dec.EmbeddedJSON(&ged.data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e event) NKeys() int {
	return 0 // parse all keys
}

var _ gojay.MarshalerJSONObject = &genericEventDetails{}
var _ EventDetails = &genericEventDetails{}

type genericEventDetails struct {
	category string
	name     string
	data     gojay.EmbeddedJSON
}

func (g genericEventDetails) Category() string {
	return g.category
}

func (g genericEventDetails) Name() string {
	return g.name
}

func (g genericEventDetails) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddEmbeddedJSON(&g.data)
}

func (g genericEventDetails) IsNil() bool {
	return false
}
