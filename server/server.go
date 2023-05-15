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
	"sync"
	"syscall"
)

type Server struct {
	nextConnectionId uint64
	logger           common.Logger
	listener         *quic.EarlyListener
	config           *Config
	qlog             qlog2.QlogWriter
	closeOnce        sync.Once
	ctx              context.Context
	cancelCtx        context.CancelFunc
}

// Run server.
// if proxyAddr is nil, no proxy is used.
func Run(addr net.UDPAddr, logPrefix string, config *Config) {
	s := &Server{
		logger:           common.DefaultLogger.WithPrefix(logPrefix),
		nextConnectionId: 0,
		config:           config,
	}
	s.ctx, s.cancelCtx = context.WithCancel(context.Background())

	tracers := make([]func(ctx context.Context, perspective logging.Perspective, connectionID logging.ConnectionID) logging.ConnectionTracer, 0)

	if s.config.QlogPathTemplate == "" {
		s.qlog = qlog2.NewStdoutQlogWriter(s.config.QlogConfig)
		tracers = append(tracers, qlog_quic.NewTracer(s.qlog))
	} else {
		tracer := qlog_quic.NewFileQlogTracer(s.config.QlogPathTemplate, s.config.QlogConfig)
		s.qlog = tracer(s.ctx, logging.PerspectiveServer, logging.ConnectionID{}).(qlog_quic.QlogWriterConnectionTracer).QlogWriter()
		tracers = append(tracers, tracer)
	}

	s.config.QuicConfig.Tracer = func(ctx context.Context, perspective logging.Perspective, id quic.ConnectionID) logging.ConnectionTracer {
		var connectionTracers []logging.ConnectionTracer
		for _, tracer := range tracers {
			connectionTracers = append(connectionTracers, tracer(ctx, perspective, id))
		}
		return logging.NewMultiplexedConnectionTracer(connectionTracers...)
	}

	//TODO add option to disable mtu discovery
	//TODO add option to enable address prevalidation

	var err error
	s.listener, err = quic.ListenAddrEarly(addr.String(), s.config.TlsConfig, s.config.QuicConfig)
	if err != nil {
		panic(err)
	}

	s.qlog.RecordEvent(qlog_app.AppInfoEvent{Message: fmt.Sprintf("starting server with pid %d, port %d", os.Getpid(), addr.Port)})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, os.Kill)
	go func() {
		<-c
		s.Close(nil)
	}()

	for {
		quicConnection, err := s.listener.Accept(context.Background())
		if err != nil {
			s.Close(err)
			return
		}
		switch alpn := quicConnection.ConnectionState().TLS.NegotiatedProtocol; alpn {
		case "qperf":
			s.acceptQperf(quicConnection)
		default:
			panic(fmt.Sprintf("unexpected ALPN: %s", alpn))
		}
	}
}

func (s *Server) Close(err error) {
	s.closeOnce.Do(func() {
		if err != nil {
			s.qlog.RecordEvent(qlog_app.AppErrorEvent{Message: err.Error()})
		}
		s.listener.Close()
		s.qlog.Close()
	})
}

func (s *Server) acceptQperf(quicConn quic.EarlyConnection) {
	_, err := newQperfConnection(
		quicConn,
		s.nextConnectionId,
		s.logger.WithPrefix(fmt.Sprintf("connection %d", s.nextConnectionId)),
		s.config,
	)
	if err != nil {
		panic(err)
	}
	s.nextConnectionId += 1
}
