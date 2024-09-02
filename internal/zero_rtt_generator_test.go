package internal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"qperf-go/common"
	"qperf-go/internal/testutils"
	"testing"
	"time"
)

func TestGenerate0RttInformation(t *testing.T) {
	var sessionTicketKey [32]byte
	var addressTokenKey [32]byte
	tokenStore, sessionCache, err := Generate0RttInformation(sessionTicketKey, addressTokenKey, "bar", "foo")
	require.NoError(t, err)
	addr, err := net.ResolveUDPAddr("udp", "[::]:0")
	conn, err := net.ListenUDP("udp", addr)
	tr := quic.Transport{
		Conn: conn,
	}
	zeroRttCounter := testutils.NewZeroRttTracer()
	pair := common.GenerateCert()
	listener, err := simpleServerFromKeys(pair, &tr, sessionTicketKey, addressTokenKey, "bar", "foo", zeroRttCounter.NewConnectionTracer)
	require.NoError(t, err)
	certPool := x509.NewCertPool()
	crt, err := x509.ParseCertificate(pair.Certificate[0])
	require.NoError(t, err)
	certPool.AddCert(crt)
	client, err := tr.DialEarly(context.Background(), listener.Addr(),
		&tls.Config{
			ClientSessionCache: sessionCache,
			RootCAs:            certPool,
			ServerName:         "bar",
			NextProtos:         []string{"foo"},
		},
		&quic.Config{
			TokenStore: tokenStore,
		},
	)
	require.NoError(t, err)
	stream, err := client.OpenStream()
	require.NoError(t, err)
	_, err = stream.Write([]byte{1, 2, 3})
	require.NoError(t, err)
	stream.Close()
	select {
	case <-zeroRttCounter.FirstByteChan():
	case <-time.After(time.Second):
	}
	client.CloseWithError(0, "")
	listener.Close()
	assert.Greater(t, zeroRttCounter.ReceivedBytes(), 0)
}
