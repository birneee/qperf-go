package common

import (
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/logging"
	"reflect"
	"unsafe"
)

//TODO find more elegant way
func GetSessionTracerByType(session quic.Session, tracerType reflect.Type) logging.ConnectionTracer {
	field := reflect.ValueOf(session).Elem().FieldByName("tracer")
	tracer := field.Elem()
	switch tracer.Type().String() {
	case "*logging.connTracerMultiplexer":
		tracers := (*[]logging.ConnectionTracer)(unsafe.Pointer(reflect.ValueOf(session).Elem().FieldByName("tracer").Elem().Elem().FieldByName("tracers").UnsafeAddr()))
		for _, tracer := range *tracers {
			if reflect.TypeOf(tracer) == tracerType {
				return tracer
			}
		}
	case tracerType.String():
		return reflect.NewAt(tracer.Elem().Type(), unsafe.Pointer(tracer.Elem().UnsafeAddr())).Interface().(logging.ConnectionTracer)
	}
	return nil
}
