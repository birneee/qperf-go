package internal

import (
	"context"
	"crypto/tls"
	"github.com/stretchr/testify/require"
	"testing"
)

func Benchmark0RTT(b *testing.B) {
	writeInitial := false
	earlySecret := false
	var key [32]byte
	_, sessionCache, err := Generate0RttInformation(key, key, "server1", "alpn1")
	require.NoError(b, err)
	conf := &tls.QUICConfig{
		TLSConfig: &tls.Config{
			MinVersion:         tls.VersionTLS13,
			ServerName:         "server1",
			ClientSessionCache: sessionCache,
			NextProtos:         []string{"alpn1"},
		},
	}
	for n := 0; n < b.N; n++ {
		conn := tls.QUICClient(conf)
		err = conn.Start(context.Background())
		require.NoError(b, err, "TLS start failed")
	loop:
		for {
			ev := conn.NextEvent()
			switch ev.Kind {
			case tls.QUICNoEvent:
				break loop
			case tls.QUICTransportParametersRequired:
				conn.SetTransportParameters(nil)
			case tls.QUICWriteData:
				switch ev.Level {
				case tls.QUICEncryptionLevelInitial:
					writeInitial = true
				default:
					panic("unexpected")
				}
			case tls.QUICSetWriteSecret:
				switch ev.Level {
				case tls.QUICEncryptionLevelEarly:
					earlySecret = true
				default:
					panic("unexpected")
				}
			default:
				panic("unexpected")
			}
		}
		require.True(b, writeInitial)
		require.True(b, earlySecret)
	}
}
