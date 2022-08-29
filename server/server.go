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
	"time"
)

// Run server.
func Run(addr net.UDPAddr, createQLog bool, tlsServerCertFile string, tlsServerKeyFile string, initialCongestionWindow uint32, minCongestionWindow uint32, maxCongestionWindow uint32, initialReceiveWindow uint64, maxReceiveWindow uint64, logPrefix string, qlogPrefix string) {

	logger := common.DefaultLogger.WithPrefix(logPrefix)

	tracers := make([]logging.Tracer, 0)

	if createQLog {
		tracers = append(tracers, common.NewQlogTrager(qlogPrefix, logger))
	}

	if initialReceiveWindow > maxReceiveWindow {
		maxReceiveWindow = initialReceiveWindow
	}

	if initialCongestionWindow < minCongestionWindow {
		initialCongestionWindow = minCongestionWindow
	}

	conf := quic.Config{
		Tracer:                         logging.NewMultiplexedTracer(tracers...),
		InitialCongestionWindow:        initialCongestionWindow,
		MinCongestionWindow:            minCongestionWindow,
		MaxCongestionWindow:            maxCongestionWindow,
		InitialStreamReceiveWindow:     initialReceiveWindow,
		MaxStreamReceiveWindow:         maxReceiveWindow,
		InitialConnectionReceiveWindow: uint64(float64(initialReceiveWindow) * quic.ConnectionFlowControlMultiplier),
		MaxConnectionReceiveWindow:     uint64(float64(maxReceiveWindow) * quic.ConnectionFlowControlMultiplier),
		//DisablePathMTUDiscovery:                          true,
		//TODO make option
		//AcceptToken: func(_ net.Addr, _ *quic.Token) bool {
		//	return true
		//},
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

	// print new reno as this is the only option in quic-go
	logger.Infof("starting server with pid %d, port %d, cc new reno, iw %d", os.Getpid(), addr.Port, conf.InitialCongestionWindow)

	var nextConnectionId uint64 = 0

	for {
		quicConnection, err := listener.Accept(context.Background())
		if err != nil {
			panic(err)
		}

		qperfSession := &qperfServerSession{
			connection:        quicConnection,
			connectionID:      nextConnectionId,
			currentRemoteAddr: quicConnection.RemoteAddr(),
			logger:            logger.WithPrefix(fmt.Sprintf("connection %d", nextConnectionId)),
		}

		go qperfSession.run()
		nextConnectionId += 1
	}
}
