package perf_integration_test

import (
	"context"
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"qperf-go/common"
	"qperf-go/perf/perf_client"
	"qperf-go/perf/perf_server"
	"testing"
	"time"
)

func TestSingleRequest(t *testing.T) {
	//os.Setenv("QLOGDIR", "tmp")
	server, err := perf_server.ListenAddr("localhost:0", &perf_server.Config{
		QuicConfig: &quic.Config{
			MaxIdleTimeout:  time.Second,
			Tracer:          qlog.DefaultConnectionTracer,
			EnableDatagrams: true,
		},
		TlsConfig: &tls.Config{
			Certificates: []tls.Certificate{common.GenerateCert()},
		},
	})
	require.NoError(t, err)
	defer server.Close()
	client, err := perf_client.DialAddr(
		server.Addr().String(),
		&perf_client.Config{
			QuicConfig: &quic.Config{
				MaxIdleTimeout:  time.Second,
				Tracer:          qlog.DefaultConnectionTracer,
				EnableDatagrams: true,
			},
			TlsConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		false,
	)
	require.NoError(t, err)
	reqStream, respStream, err := client.Request(1000, 1000, 0)
	require.NoError(t, err)
	assert.Eventually(t, func() bool {
		<-reqStream.Context().Done()
		return true
	}, time.Second, time.Millisecond)
	assert.Equal(t, context.Canceled, reqStream.Context().Err())
	assert.Equal(t, uint64(1000), reqStream.SentBytes())
	assert.Eventually(t, func() bool {
		<-respStream.Context().Done()
		return true
	}, time.Second, time.Millisecond)
	assert.Equal(t, context.Canceled, respStream.Context().Err())
	assert.True(t, respStream.Success())
	assert.Equal(t, uint64(1000), respStream.ReceivedBytes())
	err = client.Close()
	assert.NoError(t, err)
}
