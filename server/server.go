package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"net"
	"os"
	"os/signal"
	"qperf-go/common"
	qlog2 "qperf-go/common/qlog"
	"qperf-go/common/qlog_app"
	"qperf-go/perf"
	"qperf-go/perf/perf_server"
	"sync"
	"syscall"
	"time"
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
	closeOnce   sync.Once
	ctx         context.Context
	cancelCtx   context.CancelFunc
	mutex       sync.Mutex // for fields: connections
	connections map[quic.ConnectionTracingID]perf_server.Connection
	transport   quic.Transport
	// closed when client is stopping and doing some final output, goroutine waiting and cleanup
	stopping chan struct{}
}

func (s *server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *server) Context() context.Context {
	return s.ctx
}

// Listen starts server.
// if proxyAddr is nil, no proxy is used.
func Listen(addr string, config *Config) (Server, error) {
	addr = common.AppendPortIfNotSpecified(addr, perf.DefaultServerPort)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	config = config.Populate()
	s := &server{
		config:      config,
		connections: map[quic.ConnectionTracingID]perf_server.Connection{},
		stopping:    make(chan struct{}),
		transport: quic.Transport{
			Conn:                  udpConn,
			ConnectionIDGenerator: config.ConnectionIDGenerator,
			StatelessResetKey:     config.StatelessResetKey,
			TokenGeneratorKey:     config.AddressTokenKey,
		},
	}
	s.ctx, s.cancelCtx = context.WithCancel(context.Background())

	if s.qlog == nil {
		var id [4]byte
		rand.Read(id[:])
		s.qlog = qlog2.NewQlogDirWriter(id[:], s.config.PerfConfig.QlogLabel, s.config.QlogConfig)
	}

	if s.qlog == nil {
		s.qlog = qlog2.NewStdoutQlogWriter(s.config.QlogConfig)
	}
	s.config.PerfConfig.Qlog = s.qlog

	s.config.PerfConfig.QuicConfig.Tracer = appendQperfTracer(s.config.PerfConfig.QuicConfig.Tracer, s.qlog)

	//TODO add option to disable mtu discovery
	//TODO add option to enable address prevalidation

	for _, event := range s.config.Events {
		go func() {
			select {
			case <-s.stopping:
				return
			case <-time.After(event.Time()):
				s.runEvent(event)
			}
		}()
	}

	s.listener, err = s.transport.ListenEarly(s.config.PerfConfig.TlsConfig, s.config.PerfConfig.QuicConfig)
	if err != nil {
		panic(err)
	}

	s.qlog.RecordEvent(qlog_app.AppInfoEvent{Message: fmt.Sprintf("starting server with pid %d, addr %s", os.Getpid(), s.listener.Addr().String())})

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
	return s, nil
}

func appendQperfTracer(tracer func(context.Context, logging.Perspective, quic.ConnectionID) *logging.ConnectionTracer, qlog qlog2.Writer) func(context.Context, logging.Perspective, quic.ConnectionID) *logging.ConnectionTracer {
	return common.NewMultiplexedTracer(
		tracer,
		func(_ context.Context, _ logging.Perspective, _ logging.ConnectionID) *logging.ConnectionTracer {
			return &logging.ConnectionTracer{
				StartedConnection: func(_, _ net.Addr, _, destConnID logging.ConnectionID) {
					qlog.RecordEvent(common.EventConnectionStarted{DestConnectionID: destConnID})
				},
				ClosedConnection: func(err error) {
					qlog.RecordEvent(common.EventConnectionClosed{Err: err})
				},
				Debug: func(name, msg string) {
					qlog.RecordEvent(common.EventGeneric{CategoryF: "transport", NameF: name, MsgF: msg})
				},
			}
		},
	)
}

// alpn is sometimes not available immediately
// TODO fix bug in qtls
func (s *server) getAlpn(conn quic.Connection) string {
	for {
		alpn := conn.ConnectionState().TLS.NegotiatedProtocol
		if alpn != "" {
			return alpn
		}
		s.qlog.RecordEvent(qlog_app.AppInfoEvent{Message: "wait until alpn is available; TODO open quic-go github issue"})
		time.Sleep(10 * time.Microsecond)
	}
}

func (s *server) Run() error {
	for {
		quicConnection, err := s.listener.Accept(context.Background())
		if err != nil {
			s.Close(err)
			return nil
		}
		switch alpn := s.getAlpn(quicConnection); alpn {
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
		} else {
			s.qlog.RecordEvent(qlog_app.AppInfoEvent{Message: "stop"})
		}
		s.mutex.Lock()
		for _, conn := range s.connections {
			conn.Close()
		}
		s.mutex.Unlock()
		close(s.stopping)
		if s.listener != nil {
			s.listener.Close()
		}
		s.transport.Close()
		s.qlog.Close()
		s.cancelCtx()
	})
	<-s.ctx.Done()
}

func (s *server) acceptPerf(quicConn quic.EarlyConnection) {
	perfConn := perf_server.NewConnection(quicConn, s.config.PerfConfig)
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

func (s *server) runEvent(event common.Event) {
	switch event.(type) {
	default:
		panic("unexpected event type")
	}
}
