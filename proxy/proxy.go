package proxy

import (
	"crypto/tls"
	"fmt"
	"github.com/birneee/hquic-proxy-go/proxy"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/logging"
	"net"
	"qperf-go/common"
	"time"
)

// Run starts a new proxy
// nextProxyAddr the address of an additional, server-side proxy to add
// if nextProxyAddr is nil, don't add a proxy
// if clientSideInitialReceiveWindow is 0, use window from handover state
// if serverSideInitialReceiveWindow is 0, use window from handover state
func Run(addr net.UDPAddr, tlsProxyCertFile string, tlsProxyKeyFile string, nextProxyAddr *net.UDPAddr, tlsNextProxyCertFile string, clientSideInitialCongestionWindow uint32, clientSideMinCongestionWindow uint32, clientSideMaxCongestionWindow uint32, clientSideInitialReceiveWindow uint64, serverSideInitialReceiveWindow uint64, serverSideMaxReceiveWindow uint64, nextProxy0Rtt bool, qlog bool, logPrefix string) {

	controlTlsCert, err := tls.LoadX509KeyPair(tlsProxyCertFile, tlsProxyKeyFile)
	if err != nil {
		panic(err)
	}

	controlTlsConfig := &tls.Config{
		Certificates: []tls.Certificate{controlTlsCert},
	}

	controlConfig := &quic.Config{}

	var nextProxyConfig *quic.ProxyConfig
	if nextProxyAddr != nil {
		tlsConf := &tls.Config{
			RootCAs:            common.NewCertPoolWithCert(tlsNextProxyCertFile),
			ClientSessionCache: tls.NewLRUClientSessionCache(10),
			NextProtos:         []string{proxy.HQUICProxyALPN},
		}

		config := &quic.Config{
			TokenStore:           quic.NewLRUTokenStore(10, 10),
			HandshakeIdleTimeout: 10 * time.Second,
		}

		if nextProxy0Rtt {
			err := common.PingToGatherSessionTicketAndToken(nextProxyAddr.String(), tlsConf, config)
			if err != nil {
				panic(err)
			}
		}

		nextProxyConfig = &quic.ProxyConfig{
			Addr:    nextProxyAddr,
			TlsConf: tlsConf,
			Config:  config,
		}
	}

	var serverFacingTracer logging.Tracer
	var clientFacingTracer logging.Tracer
	if qlog {
		clientFacingTracer = common.NewQlogTrager(fmt.Sprintf("%s_client_facing", logPrefix))
		serverFacingTracer = common.NewQlogTrager(fmt.Sprintf("%s_server_facing", logPrefix))
	}

	clientSideProxyConf := &proxy.ProxyConnectionConfig{
		OverwriteInitialReceiveWindow: clientSideInitialReceiveWindow,
		InitialCongestionWindow:       clientSideInitialCongestionWindow,
		MinCongestionWindow:           clientSideMinCongestionWindow,
		MaxCongestionWindow:           clientSideMaxCongestionWindow,
		Tracer:                        clientFacingTracer,
	}

	serverSideProxyConf := &proxy.ProxyConnectionConfig{
		OverwriteInitialReceiveWindow: serverSideInitialReceiveWindow,
		OverwriteMaxReceiveWindow:     serverSideMaxReceiveWindow,
		Tracer:                        serverFacingTracer,
	}

	err = proxy.RunProxy(addr, controlTlsConfig, controlConfig, nextProxyConfig, clientSideProxyConf, serverSideProxyConf)
	if err != nil {
		panic(err)
	}
}
