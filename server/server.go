package server

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"net"
	"os"
	"os/signal"
	"qperf-go/common"
	qlog2 "qperf-go/common/qlog"
	"qperf-go/common/qlog_app"
	"qperf-go/common/qlog_quic"
	"qperf-go/perf"
	"qperf-go/perf/perf_server"
	"sync"
	"syscall"
)

type Server interface {
	Context() context.Context
	Close(err error)
	Addr() net.Addr
}

type server struct {
	listener    *quic.EarlyListener
	config      *Config
	qlog        qlog2.Writer
	qlogTracer  func(ctx context.Context, perspective logging.Perspective, connectionID logging.ConnectionID) logging.ConnectionTracer
	closeOnce   sync.Once
	ctx         context.Context
	cancelCtx   context.CancelFunc
	mutex       sync.Mutex // for fields below
	connections map[uint64]perf_server.Connection
}

func (s *server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *server) Context() context.Context {
	return s.ctx
}

// Listen starts server.
// if proxyAddr is nil, no proxy is used.
func Listen(addr string, config *Config) Server {
	addr = common.AppendPortIfNotSpecified(addr, perf.DefaultServerPort)
	s := &server{
		config:      config,
		connections: map[uint64]perf_server.Connection{},
	}
	s.ctx, s.cancelCtx = context.WithCancel(context.Background())

	var tracers []func(ctx context.Context, perspective logging.Perspective, connectionID logging.ConnectionID) logging.ConnectionTracer
	if s.config.QuicConfig.Tracer != nil {
		tracers = append(tracers, s.config.QuicConfig.Tracer)
	}

	if s.config.QlogPathTemplate == "" {
		s.qlog = qlog2.NewStdoutQlogWriter(s.config.QlogConfig)
		s.qlogTracer = qlog_quic.NewTracer(s.qlog)
	} else {
		s.qlogTracer = qlog_quic.NewFileQlogTracer(s.config.QlogPathTemplate, s.config.QlogConfig)
		s.qlog = s.qlogTracer(s.ctx, logging.PerspectiveServer, logging.ConnectionID{}).(qlog_quic.QlogWriterConnectionTracer).QlogWriter()
	}
	tracers = append(tracers, s.qlogTracer)

	s.config.QuicConfig.Tracer = common.NewMultiplexedTracer(tracers...)

	//TODO add option to disable mtu discovery
	//TODO add option to enable address prevalidation

	var err error
	s.listener, err = quic.ListenAddrEarly(addr, s.config.TlsConfig, s.config.QuicConfig)
	if err != nil {
		panic(err)
	}

	s.qlog.RecordEvent(qlog_app.AppInfoEvent{Message: fmt.Sprintf("starting server with pid %d, port %d", os.Getpid(), s.listener.Addr().(*net.UDPAddr).Port)})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, os.Kill)
	go func() {
		<-c
		s.Close(nil)
	}()

	go func() {
		err := s.Run()
		if err != nil {
			panic(err)
		}
	}()
	return s
}

func (s *server) Run() error {
	for {
		quicConnection, err := s.listener.Accept(context.Background())
		if err != nil {
			s.Close(err)
			return nil
		}
		switch alpn := quicConnection.ConnectionState().TLS.NegotiatedProtocol; alpn {
		case perf.ALPN:
			s.acceptPerf(quicConnection)
		default:
			panic(fmt.Sprintf("unexpected ALPN: %s", alpn))
		}
	}
}

func (s *server) Close(err error) {
	s.closeOnce.Do(func() {
		if err != nil {
			s.qlog.RecordEvent(qlog_app.AppErrorEvent{Message: err.Error()})
		}
		s.mutex.Lock()
		for _, conn := range s.connections {
			conn.Close()
		}
		s.mutex.Unlock()
		if s.listener != nil {
			s.listener.Close()
		}
		s.qlog.Close()
		s.cancelCtx()
	})
}

func (s *server) acceptPerf(quicConn quic.EarlyConnection) {
	perfConn := perf_server.NewConnection(quicConn)
	s.addConnectionToList(perfConn)
}

func (s *server) addConnectionToList(perfConn perf_server.Connection) {
	s.mutex.Lock()
	s.connections[perfConn.TracingID()] = perfConn
	s.mutex.Unlock()
	go func() {
		<-perfConn.Context().Done()
		s.mutex.Lock()
		delete(s.connections, perfConn.TracingID())
		s.mutex.Unlock()
	}()
}
