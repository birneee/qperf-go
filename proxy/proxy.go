package proxy

import (
	"crypto/tls"
	"fmt"
	"github.com/birneee/hquic-proxy-go/proxy"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/logging"
	"net"
	"os"
	"os/signal"
	"qperf-go/common"
	"time"
)

// Run starts a new proxy
// nextProxyAddr the address of an additional, server-side proxy to add
// if nextProxyAddr is nil, don't add a proxy
// if clientSideInitialReceiveWindow is 0, use window from handover state
// if serverSideInitialReceiveWindow is 0, use window from handover state
func Run(addr net.UDPAddr, tlsProxyCertFile string, tlsProxyKeyFile string, nextProxyAddr *net.UDPAddr, tlsNextProxyCertFile string, clientSideInitialCongestionWindow uint32, clientSideMinCongestionWindow uint32, clientSideMaxCongestionWindow uint32, clientSideInitialReceiveWindow uint64, serverSideInitialReceiveWindow uint64, serverSideMaxReceiveWindow uint64, nextProxy0Rtt bool, qlog bool, logPrefix string, qlogPrefix string) {

	logger := common.DefaultLogger.WithPrefix(logPrefix)

	controlTlsCert, err := tls.LoadX509KeyPair(tlsProxyCertFile, tlsProxyKeyFile)
	if err != nil {
		panic(err)
	}

	controlTlsConfig := &tls.Config{
		Certificates: []tls.Certificate{controlTlsCert},
	}

	controlConfig := &quic.Config{
		LoggerPrefix: logPrefix,
	}

	var nextProxyConfig *quic.ProxyConfig
	if nextProxyAddr != nil {
		tlsConf := &tls.Config{
			RootCAs:            common.NewCertPoolWithCert(tlsNextProxyCertFile),
			ClientSessionCache: tls.NewLRUClientSessionCache(1),
			NextProtos:         []string{proxy.HQUICProxyALPN},
		}

		config := &quic.Config{
			TokenStore:           quic.NewLRUTokenStore(1, 1),
			HandshakeIdleTimeout: 10 * time.Second,
		}

		if nextProxy0Rtt {
			err := common.PingToGatherSessionTicketAndToken(nextProxyAddr.String(), tlsConf, config)
			if err != nil {
				panic(err)
			}
		}

		nextProxyConfig = &quic.ProxyConfig{
			Addr:    nextProxyAddr.String(),
			TlsConf: tlsConf,
			Config:  config,
		}
	}

	var serverFacingTracer logging.Tracer
	var clientFacingTracer logging.Tracer
	if qlog {
		clientFacingTracer = common.NewQlogTrager(fmt.Sprintf("%s_client_facing", qlogPrefix), logger)
		serverFacingTracer = common.NewQlogTrager(fmt.Sprintf("%s_server_facing", qlogPrefix), logger)
	}

	clientSideProxyConf := &proxy.RestoreConfig{
		OverwriteInitialReceiveWindow: clientSideInitialReceiveWindow,
		InitialCongestionWindow:       clientSideInitialCongestionWindow,
		MinCongestionWindow:           clientSideMinCongestionWindow,
		MaxCongestionWindow:           clientSideMaxCongestionWindow,
		Tracer:                        clientFacingTracer,
	}

	serverSideProxyConf := &proxy.RestoreConfig{
		OverwriteInitialReceiveWindow: serverSideInitialReceiveWindow,
		MaxReceiveWindow:              serverSideMaxReceiveWindow,
		Tracer:                        serverFacingTracer,
		ProxyConf:                     nextProxyConfig,
	}

	prox, err := proxy.RunProxy(addr, controlTlsConfig, controlConfig, clientSideProxyConf, serverSideProxyConf)
	if err != nil {
		panic(err)
	}

	// close gracefully on interrupt (CTRL+C)
	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt)
	<-intChan
	_ = prox.Close()
	os.Exit(0)
}
