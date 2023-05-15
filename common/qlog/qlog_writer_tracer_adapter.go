package qlog

import (
	"context"
	"github.com/francoispqt/gojay"
	"github.com/quic-go/quic-go/logging"
	"github.com/quic-go/quic-go/qlog"
	"reflect"
	"time"
)

// NewQlogWriterTracerAdapter allows to use the QlogWriter as a default quic-go connection tracer.
// multi connection tracers can write to the same QlogWriter.
// this significantly increases cpu usage, and degrades transport performance.
func NewQlogWriterTracerAdapter(qlogWriter QlogWriter) func(context.Context, logging.Perspective, logging.ConnectionID) logging.ConnectionTracer {

	return func(ctx context.Context, p logging.Perspective, id logging.ConnectionID) logging.ConnectionTracer {
		tracer := qlog.NewConnectionTracer(&childWriter{qlogWriter: qlogWriter}, p, id)
		hackReferenceTime(tracer, qlogWriter.ReferenceTime())
		return tracer
	}
}

// set (hack) reference time to quic-go/qlog/connectionTracer
func hackReferenceTime(tracer logging.ConnectionTracer, referenceTime time.Time) {
	o := reflect.ValueOf(tracer).Elem()
	time := (*time.Time)(o.FieldByName("referenceTime").Addr().UnsafePointer())
	*time = referenceTime
}

type childWriter struct {
	qlogWriter          QlogWriter
	readTopLevelElement bool
	odcid               string
	groupID             string
}

func (c *childWriter) Write(p []byte) (int, error) {
	if p[0] != '{' {
		return len(p), nil // skip non-json writes
	}
	if !c.readTopLevelElement {
		tl := topLevel{}
		err := gojay.UnmarshalJSONObject(p, &tl)
		if err != nil {
			return 0, err
		}
		c.odcid = tl.trace.CommonFields.ODCID
		c.groupID = tl.trace.CommonFields.GroupID
		c.readTopLevelElement = true
		return len(p), nil
	}
	e := &event{}
	err := gojay.UnmarshalJSONObject(p, e)
	if err != nil {
		return 0, err
	}
	e.GroupID = c.groupID
	c.qlogWriter.RecordEventWithTimeGroupODCID(e.EventDetails, c.qlogWriter.ReferenceTime().Add(e.RelativeTime), c.groupID, c.odcid)
	return len(p), nil
}

func (c *childWriter) Close() error {
	// do nothing
	return nil
}
