package common

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/quic-go/quic-go"
)

// PingToGatherSessionTicketAndToken establishes a new QUIC connection.
// As soon as the session ticket and the token is received, the connection is closed.
// This function can be used to prepare for 0-RTT
// TODO add timeout
func PingToGatherSessionTicketAndToken(ctx context.Context, addr string, tlsConf *tls.Config, config *quic.Config) error {
	if tlsConf.ClientSessionCache == nil {
		return errors.New("session cache is nil")
	}
	if config.TokenStore == nil {
		panic("session cache is nil")
	}

	singleSessionCache := NewSingleSessionCache()
	singleTokenStore := NewSingleTokenStore()

	tmpTlsConf := tlsConf.Clone()
	tmpTlsConf.ClientSessionCache = singleSessionCache
	tmpConfig := config.Clone()
	tmpConfig.TokenStore = singleTokenStore

	connection, err := quic.DialAddr(ctx, addr, tmpTlsConf, tmpConfig)
	if err != nil {
		return err
	}

	tlsConf.ClientSessionCache.Put(singleSessionCache.Await())
	config.TokenStore.Put(singleTokenStore.Await())

	err = connection.CloseWithError(quic.ApplicationErrorCode(0), "cancel")
	if err != nil {
		return err
	}
	return nil
}
