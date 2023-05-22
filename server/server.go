package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"net"
	"os"
	"os/signal"
	"qperf-go/common"
	qlog2 "qperf-go/common/qlog"
	"qperf-go/common/qlog_app"
	"qperf-go/common/qlog_hquic"
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
	nextConnectionId        uint64
	logger                  common.Logger
	listener                *quic.EarlyListener
	config                  *Config
	qlog                    qlog2.QlogWriter
	qlogTracer              func(ctx context.Context, perspective logging.Perspective, connectionID logging.ConnectionID) logging.ConnectionTracer
	closeOnce               sync.Once
	ctx                     context.Context
	cancelCtx               context.CancelFunc
	queuesPacketsForRestore *common.EncryptedPacketQueues
	mutex                   sync.Mutex // for fields below
	// key is ODCID
	connections map[quic.ConnectionID]*qperfServerSession
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
		logger:                  common.DefaultLogger.WithPrefix(logPrefix),
		nextConnectionId:        0,
		config:                  config,
		queuesPacketsForRestore: common.NewEncryptedPacketQueues(),
		//TODO remove closed connections from map
		connections: map[quic.ConnectionID]*qperfServerSession{},
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

	if s.config.StateServer != nil {
		s.config.QuicConfig.HandleUnknownConnectionPacket = s.handleUnknownConnectionID
		s.config.StateTransferConfig.QuicConfig.TokenStore = quic.NewLRUTokenStore(1, 1)
		s.config.StateTransferConfig.QuicConfig.Tracer = common.NewMultiplexedTracer(s.qlogTracer)
		s.config.StateTransferConfig.TlsConfig.ClientSessionCache = tls.NewLRUClientSessionCache(1)

		if s.config.Use0RTTStateRequest {
			err := common.PingToGatherSessionTicketAndToken(context.Background(), s.config.StateServer.String(), s.config.StateTransferConfig.TlsConfig, s.config.StateTransferConfig.QuicConfig)
			if err != nil {
				msg := fmt.Sprintf("failed to store session ticket and address token of state server for 0-RTT: %s", err)
				s.qlog.RecordEvent(qlog_app.AppErrorEvent{Message: msg})
				s.Close(fmt.Errorf(msg))
				return s
			}
			s.qlog.RecordEvent(qlog_app.AppInfoEvent{Message: "stored session ticket and address token of state server for 0-RTT"})
		}
	}

	if s.config.ServeState {
		s.config.TlsConfig.NextProtos = append(s.config.TlsConfig.NextProtos, quic.HQUICStateTransferALPN)
	}

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
		case quic.HQUICStateTransferALPN:
			err := s.acceptHquic(quicConnection)
			if err != nil {
				fmt.Printf("error on hquic connection: %s\n", err.Error())
			}
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
		if s.listener != nil {
			s.listener.Close()
		}
		s.mutex.Lock()
		for _, conn := range s.connections {
			conn.quicConn.CloseWithError(errors.NoError, "no error")
		}
		s.mutex.Unlock()
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
	s.connections[quicConn.OriginalDestinationConnectionID()] = conn
	s.nextConnectionId += 1
}

func (s *server) acceptHquic(quicConn quic.EarlyConnection) error {
	hquicConn := quic.NewStateTransferConnection(quicConn)
	requestedConnID, err := hquicConn.ReceiveRequest()
	if err != nil {
		return err
	}
	s.qlog.RecordEvent(qlog_hquic.StateRequestReceivedEvent{})
	fmt.Printf("received state request for %s\n", requestedConnID.String())
	connToSerialize := s.listener.PacketHandlerManager().GetConnectionByID(requestedConnID)
	if connToSerialize == nil {
		hquicConn.CloseWithError(quic.HquicErrorCodesUnknownConnectionID, "unknown connection id")
	}
	qperfConn, ok := s.connections[connToSerialize.OriginalDestinationConnectionID()]
	if !ok {
		panic("no qperf connection associated with ODCID")
	}
	qperfState, err := qperfConn.Handover()
	if err != nil {
		panic(err)
	}
	s.qlog.RecordEvent(qlog_hquic.StateStoredEvent{})

	serializedState, err := qperfState.Serialize()
	if err != nil {
		panic(err)
	}
	s.qlog.RecordEvent(qlog_hquic.StateSerializedEvent{})
	err = hquicConn.SendState(serializedState)
	if err != nil {
		panic(err)
	}
	s.qlog.RecordEvent(qlog_hquic.StateSentEvent{PayloadBytes: len(serializedState)})
	fmt.Printf("sent state: %s\n", serializedState)
	return nil
}

func (s *server) handleUnknownConnectionID(connID quic.ConnectionID, packet quic.UnhandledPacket) {

	s.qlog.RecordEvent(qlog_hquic.UnknownConnectionPacketReceived{})
	newConnID := s.queuesPacketsForRestore.Enqueue(connID, packet)

	if newConnID {
		conn, err := quic.DialStateTransfer(context.Background(), s.config.StateServer.String(), s.config.StateTransferConfig)
		if err != nil {
			panic(err)
		}
		err = conn.SendRequest(connID)
		if err != nil {
			panic(err)
		}
		s.qlog.RecordEvent(qlog_hquic.StateRequestSentEvent{})
		go func() {
			defer conn.CloseWithError(quic.HquicErrorCodesNoError, "no error")
			serializedState, err := conn.ReceiveState()
			if err != nil {
				panic(err)
			}
			s.qlog.RecordEvent(qlog_hquic.StateReceivedEvent{})
			fmt.Printf("received state: %s\n", serializedState)

			state, err := (&ConnectionState{}).Parse(serializedState)
			if err != nil {
				panic(err)
			}
			s.qlog.RecordEvent(qlog_hquic.StateParsedEvent{})

			qperfConn, err := restoreQperfConnection(state, s.listener, s.nextConnectionId, s.logger.WithPrefix(fmt.Sprintf("connection %d", s.nextConnectionId)), s.config.QuicConfig.Tracer, s.config)
			if err != nil {
				panic(err)
			}
			s.qlog.RecordEvent(qlog_hquic.StateRestoredEvent{})
			s.connections[qperfConn.quicConn.OriginalDestinationConnectionID()] = qperfConn
			s.nextConnectionId++

			s.queuesPacketsForRestore.Dequeue(connID, qperfConn.quicConn)
		}()
	}
}
