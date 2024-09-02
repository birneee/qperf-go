package internal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"net"
	"qperf-go/common"
)

func simpleServerFromKeys(crt tls.Certificate, t *quic.Transport, sessionTicketKey [32]byte, addressTokenKey quic.TokenGeneratorKey, serverName string, alpn string, tracer func(ctx context.Context, perspective logging.Perspective, id quic.ConnectionID) *logging.ConnectionTracer) (*quic.EarlyListener, error) {
	if t.TokenGeneratorKey != nil {
		panic("")
	}
	t.TokenGeneratorKey = &addressTokenKey
	quicConfig := &quic.Config{
		Allow0RTT: true,
		Tracer:    tracer,
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{crt},
		NextProtos:   []string{alpn},
		ServerName:   serverName,
	}
	tlsConfig.SetSessionTicketKeys([][32]byte{sessionTicketKey})
	listener, err := t.ListenEarly(
		tlsConfig,
		quicConfig,
	)
	if err != nil {
		return nil, err
	}
	return listener, nil
}

// Generate0RttInformation generates an address token and a session ticket without opening a connection to actual. server.
// Allows to make 0-RTT Handshakes without previous connections.
// The certificate of the servers can be different.
// SessionTicketKey, addressTokenKey, and ALPN must match on the actual server.
// TODO transport parameters?
func Generate0RttInformation(sessionTicketKey [32]byte, addressTokenKey [32]byte, serverName string, alpn string) (quic.TokenStore, tls.ClientSessionCache, error) {
	addr, err := net.ResolveUDPAddr("udp", "[::]:0")
	conn, err := net.ListenUDP("udp", addr)
	t := quic.Transport{
		Conn: conn,
	}
	keyPair := common.GenerateCertFor([]string{serverName}, nil)
	listener, err := simpleServerFromKeys(keyPair, &t, sessionTicketKey, addressTokenKey, serverName, alpn, nil)
	if err != nil {
		return nil, nil, err
	}
	sessionCache := common.NewSingleSessionCache()
	tokenStore := common.NewSingleTokenStore()
	certPool := x509.NewCertPool()
	cert, err := x509.ParseCertificate(keyPair.Certificate[0])
	certPool.AddCert(cert)
	client, err := t.DialEarly(context.Background(), listener.Addr(),
		&tls.Config{
			ClientSessionCache: sessionCache,
			NextProtos:         []string{alpn},
			RootCAs:            certPool,
			ServerName:         serverName,
		},
		&quic.Config{
			TokenStore: tokenStore,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	sessionCache.Await()
	tokenStore.Await()
	client.CloseWithError(0, "")
	listener.Close()
	return tokenStore, sessionCache, nil
}
