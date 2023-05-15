package qlog_app

import "github.com/francoispqt/gojay"

type AppInfoEvent struct {
	Message string
}

func (e AppInfoEvent) Category() string { return "app" }
func (e AppInfoEvent) Name() string     { return "info" }
func (e AppInfoEvent) IsNil() bool      { return false }

func (e AppInfoEvent) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey("message", e.Message)
}

type AppErrorEvent struct {
	Message string
}

func (e AppErrorEvent) Category() string { return "app" }
func (e AppErrorEvent) Name() string     { return "error" }
func (e AppErrorEvent) IsNil() bool      { return false }

func (e AppErrorEvent) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey("message", e.Message)
}
