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
// if proxyAddr is nil, no proxy is used.
func Run(addr net.UDPAddr, createQLog bool, migrateAfter time.Duration, tlsServerCertFile string, tlsServerKeyFile string, initialCongestionWindow uint32, minCongestionWindow uint32, maxCongestionWindow uint32, initialReceiveWindow uint64, maxReceiveWindow uint64, noXse bool, logPrefix string, qlogPrefix string) {

	logger := common.DefaultLogger.WithPrefix(logPrefix)

	tracers := make([]logging.Tracer, 0)

	if createQLog {
		tracers = append(tracers, common.NewQlogTracer(qlogPrefix, logger))
	}

	//TODO somehow associate it with the qperf session for logging
	tracers = append(tracers, common.NewEventTracer(common.Handlers{
		UpdatePath: func(odcid logging.ConnectionID, newRemote net.Addr) {
			logger.Infof("migrated QUIC connection %s to %s", odcid.String(), newRemote)
		},
		StartedConnection: func(odcid logging.ConnectionID, local, remote net.Addr, srcConnID, destConnID logging.ConnectionID) {
			logger.Infof("started QUIC connection %s", odcid.String())
		},
		ClosedConnection: func(odcid logging.ConnectionID, err error) {
			logger.Infof("closed QUIC connection %s", odcid.String())
		},
	}))

	if initialReceiveWindow > maxReceiveWindow {
		maxReceiveWindow = initialReceiveWindow
	}

	if initialCongestionWindow < minCongestionWindow {
		initialCongestionWindow = minCongestionWindow
	}

	conf := quic.Config{
		Tracer:                         logging.NewMultiplexedTracer(tracers...),
		EnableActiveMigration:          true,
		InitialCongestionWindow:        initialCongestionWindow,
		MinCongestionWindow:            minCongestionWindow,
		MaxCongestionWindow:            maxCongestionWindow,
		InitialStreamReceiveWindow:     initialReceiveWindow,
		MaxStreamReceiveWindow:         maxReceiveWindow,
		InitialConnectionReceiveWindow: uint64(float64(initialReceiveWindow) * quic.ConnectionFlowControlMultiplier),
		MaxConnectionReceiveWindow:     uint64(float64(maxReceiveWindow) * quic.ConnectionFlowControlMultiplier),
		//TODO add option to disable mtu discovery
		//TODO add option to enable address prevalidation
	}

	if noXse {
		conf.ExtraStreamEncryption = quic.DisableExtraStreamEncryption
	} else {
		conf.ExtraStreamEncryption = quic.PreferExtraStreamEncryption
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

	// migrate
	if migrateAfter.Nanoseconds() != 0 {
		go func() {
			time.Sleep(migrateAfter)
			addr, err := listener.MigrateUDPSocket()
			if err != nil {
				panic(err)
			}
			logger.Infof("migrated to %s", addr.String())
		}()
	}

	var nextConnectionId uint64 = 0

	for {
		quicConnection, err := listener.Accept(context.Background())
		if err != nil {
			panic(err)
		}

		qperfSession := &qperfServerSession{
			connection:   quicConnection,
			connectionID: nextConnectionId,
			logger:       logger.WithPrefix(fmt.Sprintf("connection %d", nextConnectionId)),
		}

		go qperfSession.run()
		nextConnectionId += 1
	}
}
