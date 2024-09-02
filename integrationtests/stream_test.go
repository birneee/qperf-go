package integrationtests

import (
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"github.com/quic-go/quic-go/qlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"qperf-go/client"
	"qperf-go/common"
	"qperf-go/perf/perf_server"
	"qperf-go/server"
	"testing"
	"time"
)

type TestingT interface {
	Errorf(format string, args ...interface{})
	FailNow()
	Cleanup(f func())
}

func newSimpleTestServer(t TestingT) server.Server {
	server, err := server.Listen("localhost:0", &server.Config{
		PerfConfig: &perf_server.Config{
			TlsConfig: &tls.Config{
				Certificates: []tls.Certificate{common.GenerateCert()},
			},
			QuicConfig: &quic.Config{
				MaxIdleTimeout: time.Second,
				Tracer:         qlog.DefaultConnectionTracer,
			},
			QlogLabel: "qperf_server",
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Close(nil)
	})
	return server
}

func TestSingleRequest(t *testing.T) {
	//os.Setenv("QLOGDIR", "tmp")
	server := newSimpleTestServer(t)
	client := client.Dial(&client.Config{
		RemoteAddress:  server.Addr().String(),
		RequestLength:  100_000,
		ResponseLength: 100_000,
		NumRequests:    1,
		QuicConfig: &quic.Config{
			MaxIdleTimeout:  time.Second,
			EnableDatagrams: true,
		},
		TlsConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	<-client.Context().Done()
	report := client.TotalReport()
	assert.Equal(t, logging.ByteCount(100_000), report.ReceivedBytes)
	assert.Equal(t, logging.ByteCount(100_000), report.SentBytes)
}

func Test1SecProbeTime(t *testing.T) {
	//os.Setenv("QLOGDIR", "tmp")
	server := newSimpleTestServer(t)
	client := client.Dial(&client.Config{
		RemoteAddress:   server.Addr().String(),
		RequestLength:   100_000,
		ResponseLength:  100_000,
		RequestInterval: time.Second / 10,
		ProbeTime:       time.Second,
		QuicConfig: &quic.Config{
			MaxIdleTimeout:  time.Second,
			EnableDatagrams: true,
		},
		TlsConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	<-client.Context().Done()
	report := client.TotalReport()
	assert.Equal(t, logging.ByteCount(1_000_000), report.ReceivedBytes)
	assert.Equal(t, logging.ByteCount(1_000_000), report.SentBytes)
}
