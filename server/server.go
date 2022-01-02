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
func Run(addr net.UDPAddr, createQLog bool, migrateAfter time.Duration, proxyAddr *net.UDPAddr, tlsServerCertFile string, tlsServerKeyFile string, tlsProxyCertFile string, initialCongestionWindow uint32, minCongestionWindow uint32, maxCongestionWindow uint32, initialReceiveWindow uint64, maxReceiveWindow uint64) {

	// TODO
	if proxyAddr != nil {
		panic("implement me")
	}

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

	if initialCongestionWindow < minCongestionWindow {
		initialCongestionWindow = minCongestionWindow
	}

	conf := quic.Config{
		Tracer: logging.NewMultiplexedTracer(tracers...),
		IgnoreReceived1RTTPacketsUntilFirstPathMigration: proxyAddr != nil,
		EnableActiveMigration:                            true,
		InitialCongestionWindow:                          initialCongestionWindow,
		MinCongestionWindow:                              minCongestionWindow,
		MaxCongestionWindow:                              maxCongestionWindow,
		InitialStreamReceiveWindow:                       initialReceiveWindow,
		MaxStreamReceiveWindow:                           maxReceiveWindow,
		InitialConnectionReceiveWindow:                   uint64(float64(initialReceiveWindow) * quic.ConnectionFlowControlMultiplier),
		MaxConnectionReceiveWindow:                       uint64(float64(maxReceiveWindow) * quic.ConnectionFlowControlMultiplier),
		//DisablePathMTUDiscovery:                          true,
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

	logger.Infof("starting server with pid %d, port %d, cc cubic, iw %d", os.Getpid(), addr.Port, conf.InitialCongestionWindow)

	// migrate
	if migrateAfter.Nanoseconds() != 0 {
		go func() {
			time.Sleep(migrateAfter)
			addr, err := listener.MigrateUDPSocket()
			if err != nil {
				panic(err)
			}
			fmt.Printf("migrated to %s\n", addr.String())
		}()
	}

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
