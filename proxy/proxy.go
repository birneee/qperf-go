package proxy

import (
	"crypto/tls"
	"github.com/birneee/hquic-proxy-go/proxy"
	"github.com/lucas-clemente/quic-go"
	"net"
	"qperf-go/common"
)

// Run starts a new proxy
// nextProxyAddr the address of an additional, server-side proxy to add
// if nextProxyAddr is nil, don't add a proxy
// if clientSideInitialReceiveWindow is 0, use window from handover state
// if serverSideInitialReceiveWindow is 0, use window from handover state
func Run(addr net.UDPAddr, tlsProxyCertFile string, tlsProxyKeyFile string, nextProxyAddr *net.UDPAddr, tlsNextProxyCertFile string, initialCongestionWindow uint32, clientSideInitialReceiveWindow uint64, serverSideInitialReceiveWindow uint64, serverSideMaxReceiveWindow uint64, nextProxy0Rtt bool) {

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
			TokenStore: quic.NewLRUTokenStore(10, 10),
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

	overwriteConfig := &proxy.HandoverOverwriteConfig{
		ClientSideInitialReceiveWindow:    clientSideInitialReceiveWindow,
		ServerSideInitialReceiveWindow:    serverSideInitialReceiveWindow,
		ServerSideMaxReceiveWindow:        serverSideMaxReceiveWindow,
		ClientSideInitialCongestionWindow: initialCongestionWindow,
	}

	err = proxy.RunProxy(addr, controlTlsConfig, controlConfig, nextProxyConfig, overwriteConfig)
	if err != nil {
		panic(err)
	}
}
