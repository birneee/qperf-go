package perf_integration_test

import (
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/assert"
	"qperf-go/common"
	"qperf-go/perf/perf_client"
	"qperf-go/perf/perf_server"
	"testing"
	"time"
)

func Test(t *testing.T) {
	const responseLength = 1000
	clientConfig := perf_client.Config{
		QuicConfig: &quic.Config{
			MaxIdleTimeout: 100 * time.Millisecond,
		},
		TlsConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	common.GenerateCert()
	server, err := perf_server.ListenAddr("localhost:0", &perf_server.Config{
		QuicConfig: &quic.Config{
			MaxIdleTimeout: 100 * time.Millisecond,
		},
		TlsConfig: &tls.Config{
			Certificates: []tls.Certificate{common.GenerateCert()},
		},
	})
	if !assert.NoError(t, err) {
		return
	}
	client, err := perf_client.DialAddr(server.Addr().String(), &clientConfig)
	if !assert.NoError(t, err) {
		return
	}
	reqStream, respStream, err := client.Request(1000, responseLength, 0)
	if !assert.NoError(t, err) {
		return
	}
	assert.Eventually(t, func() bool {
		return reqStream.SentBytes() == 1012
	}, time.Second, time.Millisecond)
	assert.Eventually(t, func() bool {
		return respStream.ReceivedBytes() == 1000
	}, time.Second, time.Millisecond)
	client.Close()
	server.Close()
}
