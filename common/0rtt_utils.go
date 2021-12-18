package common

import (
	"crypto/tls"
	"errors"
	"github.com/lucas-clemente/quic-go"
	"net"
	"time"
)

// PingToGatherSessionTicketAndToken establishes a new QUIC connection.
// As soon as the session ticket and the token is received, the connection is closed.
// This function can be used to prepare for 0-RTT
func PingToGatherSessionTicketAndToken(addr string, tlsConf *tls.Config, config *quic.Config) error {
	if tlsConf.ClientSessionCache == nil {
		return errors.New("session cache is nil")
	}
	if config.TokenStore == nil {
		panic("session cache is nil")
	}
	session, err := quic.DialAddr(addr, tlsConf, config)
	if err != nil {
		return err
	}

	sessionCacheKey := sessionCacheKey(session.RemoteAddr(), tlsConf)
	tokenStoreKey := tokenStoreKey(session.RemoteAddr(), tlsConf)

	// await session ticket
	for {
		time.Sleep(time.Millisecond)
		_, ok := tlsConf.ClientSessionCache.Get(sessionCacheKey)
		if ok {
			break
		}
	}
	// await token
	for {
		time.Sleep(time.Millisecond)
		token := config.TokenStore.Pop(tokenStoreKey)
		if token != nil {
			config.TokenStore.Put(tokenStoreKey, token) // put back again
			break
		}
	}
	err = session.CloseWithError(quic.ApplicationErrorCode(0), "cancel")
	if err != nil {
		return err
	}
	return nil
}

// inspired by qtls.clientSessionCacheKey implementation
// TODO avoid duplicate code
func sessionCacheKey(serverAddr net.Addr, tlsConf *tls.Config) string {
	if len(tlsConf.ServerName) > 0 {
		return "qtls-" + tlsConf.ServerName
	}
	return "qtls-" + serverAddr.String()
}

// inspired by quic.newClientSession implementation
// TODO avoid duplicate code
func tokenStoreKey(serverAddr net.Addr, tlsConf *tls.Config) string {
	if len(tlsConf.ServerName) > 0 {
		return tlsConf.ServerName
	} else {
		return serverAddr.String()
	}
}
