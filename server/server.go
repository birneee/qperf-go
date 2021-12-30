package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/logging"
	"net"
	"os"
	"qperf-go/common"
)

// Run server.
func Run(addr net.UDPAddr, createQLog bool, tlsServerCertFile string, tlsServerKeyFile string, initialReceiveWindow uint64, maxReceiveWindow uint64) {

	state := common.State{}

	tracers := make([]logging.Tracer, 0)

	tracers = append(tracers, common.StateTracer{
		State: &state,
	})

	if createQLog {
		tracers = append(tracers, common.NewQlogTrager("server"))
	}

	if initialReceiveWindow > maxReceiveWindow {
		maxReceiveWindow = initialReceiveWindow
	}

	conf := quic.Config{
		Tracer:                         logging.NewMultiplexedTracer(tracers...),
		InitialStreamReceiveWindow:     initialReceiveWindow,
		MaxStreamReceiveWindow:         maxReceiveWindow,
		InitialConnectionReceiveWindow: uint64(float64(initialReceiveWindow) * common.ConnectionFlowControlMultiplier),
		MaxConnectionReceiveWindow:     uint64(float64(maxReceiveWindow) * common.ConnectionFlowControlMultiplier),
	}

	//TODO make CLI option
	tlsCert, err := tls.LoadX509KeyPair(tlsServerCertFile, tlsServerKeyFile)
	if err != nil {
		panic(err)
	}

	tlsConf := tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"qperf"},
	}

	listener, err := quic.ListenAddrEarly(addr.String(), &tlsConf, &conf)
	if err != nil {
		panic(err)
	}

	logger := common.DefaultLogger.Clone()
	if len(os.Getenv(common.LogEnv)) == 0 {
		logger.SetLogLevel(common.LogLevelInfo) // log level info is the default
	}

	logger.Infof("starting server with pid %d, port %d, cc cubic, iw %d", os.Getpid(), addr.Port, common.InitialCongestionWindow)

	var nextSessionId uint64 = 0

	for {
		quicSession, err := listener.Accept(context.Background())
		if err != nil {
			panic(err)
		}

		qperfSession := &qperfServerSession{
			session:           quicSession,
			sessionID:         nextSessionId,
			currentRemoteAddr: quicSession.RemoteAddr(),
			logger:            logger.WithPrefix(fmt.Sprintf("session %d", nextSessionId)),
		}

		go qperfSession.run()
		nextSessionId += 1
	}
}
