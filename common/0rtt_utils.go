package common

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
)

// PingToGatherSessionTicketAndToken establishes a new QUIC connection.
// As soon as the session ticket and the token is received, the connection is closed.
// This function can be used to prepare for 0-RTT
// TODO add timeout
func PingToGatherSessionTicketAndToken(
	ctx context.Context,
	addr string,
	sessionCache tls.ClientSessionCache,
	tokenStore quic.TokenStore,
	alpn string,
	certPool *x509.CertPool,
	serverName string,
	tracer func(context.Context, logging.Perspective, logging.ConnectionID) *logging.ConnectionTracer,
) error {
	tlsConf := &tls.Config{
		ClientSessionCache: NewSingleSessionCache(),
		NextProtos:         []string{alpn},
		RootCAs:            certPool,
		ServerName:         serverName,
	}
	quicConf := &quic.Config{
		TokenStore: NewSingleTokenStore(),
		Tracer:     tracer,
	}

	connection, err := quic.DialAddr(ctx, addr, tlsConf, quicConf)
	if err != nil {
		return err
	}

	sessionCache.Put(tlsConf.ClientSessionCache.(*SingleSessionCache).Await())
	tokenStore.Put(quicConf.TokenStore.(*SingleTokenStore).Await())

	err = connection.CloseWithError(quic.ApplicationErrorCode(0), "cancel")
	if err != nil {
		return err
	}
	return nil
}
