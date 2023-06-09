package qlog

// inspired by quic-go/qlog/qlog.go

import (
	"fmt"
	"github.com/francoispqt/gojay"
	"io"
	"log"
	"sync"
	"time"
)

type QlogWriter interface {
	RecordEvent(details EventDetails)
	ReferenceTime() time.Time
	RecordEventAtTime(time time.Time, details EventDetails)
	RecordEventAtTimeWithGroup(details EventDetails, time time.Time, groupID string)
	RecordEventWithTimeGroupODCID(details EventDetails, time time.Time, groupID string, odcid string)
	Close()
	Includes(category string, name string) bool
	Config() Config
}

const eventChanSize = 50

type qlogWriter struct {
	mutex sync.Mutex

	w             io.WriteCloser
	referenceTime time.Time

	events     chan event
	encodeErr  error
	runStopped chan struct{}
	config     *Config
}

func (w *qlogWriter) Config() Config {
	return *w.config.Copy()
}

func (w *qlogWriter) Includes(category string, name string) bool {
	return w.config.Included(category, name)
}

func NewQlogWriter(wc io.WriteCloser, config *Config) QlogWriter {
	w := &qlogWriter{
		w:             wc,
		runStopped:    make(chan struct{}),
		events:        make(chan event, eventChanSize),
		referenceTime: time.Now(),
		config:        config,
	}
	go w.run()
	return w
}

func (w *qlogWriter) run() {
	defer close(w.runStopped)
	enc := gojay.NewEncoder(w.w)
	tl := &topLevel{
		trace: trace{
			Title:       w.config.Title,
			CodeVersion: w.config.CodeVersion,
			VantagePoint: vantagePoint{
				Type: w.config.VantagePoint,
			},
			CommonFields: commonFields{
				ReferenceTime: w.referenceTime,
				GroupID:       w.config.GroupID,
				ODCID:         w.config.ODCID,
			},
		},
	}
	if err := enc.Encode(tl); err != nil {
		panic(fmt.Sprintf("qlog encoding failed: %s", err))
	}
	if _, err := w.w.Write([]byte{'\n'}); err != nil {
		panic(fmt.Sprintf("qlog encoding failed: %s", err))
	}
	for ev := range w.events {
		if w.encodeErr != nil { // if encoding failed, just continue draining the event channel
			continue
		}
		if !w.Includes(ev.EventDetails.Category(), ev.EventDetails.Name()) {
			continue
		}
		if err := enc.Encode(ev); err != nil {
			w.encodeErr = err
			continue
		}
		if _, err := w.w.Write([]byte{'\n'}); err != nil {
			w.encodeErr = err
		}
	}
}

func (w *qlogWriter) Close() {
	if err := w.export(); err != nil {
		log.Printf("exporting qlog failed: %s\n", err)
	}
}

// export writes a qlog.
func (w *qlogWriter) export() error {
	close(w.events)
	<-w.runStopped
	if w.encodeErr != nil {
		return w.encodeErr
	}
	return w.w.Close()
}

func (w *qlogWriter) recordEvent(event event) {
	w.events <- event
}

func (w *qlogWriter) RecordEvent(details EventDetails) {
	w.mutex.Lock()
	w.RecordEventAtTime(time.Now(), details)
	w.mutex.Unlock()
}

func (w *qlogWriter) RecordEventAtTime(time time.Time, details EventDetails) {
	w.recordEvent(event{
		RelativeTime: time.Sub(w.referenceTime),
		EventDetails: details,
	})
}

func (w *qlogWriter) RecordEventAtTimeWithGroup(details EventDetails, time time.Time, groupID string) {
	if groupID == w.config.GroupID {
		w.recordEvent(event{
			RelativeTime: time.Sub(w.referenceTime),
			EventDetails: details,
		})
	} else {
		w.recordEvent(event{
			RelativeTime: time.Sub(w.referenceTime),
			EventDetails: details,
			GroupID:      groupID,
		})
	}
}

func (w *qlogWriter) RecordEventWithTimeGroupODCID(details EventDetails, time time.Time, groupID string, odcid string) {
	if groupID == w.config.GroupID {
		w.recordEvent(event{
			RelativeTime: time.Sub(w.referenceTime),
			EventDetails: details,
		})
	} else {
		w.recordEvent(event{
			RelativeTime: time.Sub(w.referenceTime),
			EventDetails: details,
			GroupID:      groupID,
			ODCID:        odcid,
		})
	}
}

func (w *qlogWriter) ReferenceTime() time.Time {
	return w.referenceTime
}
