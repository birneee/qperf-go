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
	"qperf-go/errors"
	"sync"
	"syscall"
)

type Server interface {
	Context() context.Context
	Close(err error)
	Addr() net.Addr
}

type server struct {
	nextConnectionId uint64
	logger           common.Logger
	listener         *quic.EarlyListener
	config           *Config
	qlog             qlog2.QlogWriter
	qlogTracer       func(ctx context.Context, perspective logging.Perspective, connectionID logging.ConnectionID) logging.ConnectionTracer
	closeOnce        sync.Once
	ctx              context.Context
	cancelCtx        context.CancelFunc
	mutex            sync.Mutex // for fields below
	//TODO remove closed connections from map
	connections []*qperfServerSession
}

func (s *server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *server) Context() context.Context {
	return s.ctx
}

// Run server.
// if proxyAddr is nil, no proxy is used.
func Listen(addr string, logPrefix string, config *Config) Server {
	addr = common.AppendPortIfNotSpecified(addr, common.DefaultQperfServerPort)
	s := &server{
		logger:           common.DefaultLogger.WithPrefix(logPrefix),
		nextConnectionId: 0,
		config:           config,
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
		case common.QperfALPN:
			s.acceptQperf(quicConnection)
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
			conn.quicConn.CloseWithError(errors.NoError, "no error")
		}
		s.mutex.Unlock()
		if s.listener != nil {
			s.listener.Close()
		}
		s.qlog.Close()
		s.cancelCtx()
	})
}

func (s *server) acceptQperf(quicConn quic.EarlyConnection) {
	conn, err := newQperfConnection(
		quicConn,
		s.nextConnectionId,
		s.logger.WithPrefix(fmt.Sprintf("connection %d", s.nextConnectionId)),
		s.config,
	)
	if err != nil {
		panic(err)
	}
	s.mutex.Lock()
	s.connections = append(s.connections, conn)
	s.mutex.Unlock()
	s.nextConnectionId += 1
}
