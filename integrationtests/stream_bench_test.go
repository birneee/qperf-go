package integrationtests

import (
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"github.com/stretchr/testify/assert"
	"qperf-go/client"
	"testing"
	"time"
)

func BenchmarkBulkData(b *testing.B) {
	//os.Setenv("QLOGDIR", "tmp")
	server := newSimpleTestServer(b)
	client := client.Dial(&client.Config{
		RemoteAddress:  server.Addr().String(),
		RequestLength:  uint64(b.N),
		ResponseLength: uint64(b.N),
		QuicConfig: &quic.Config{
			MaxIdleTimeout: time.Second,
		},
		TlsConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	<-client.Context().Done()
	report := client.TotalReport()
	assert.Equal(b, logging.ByteCount(b.N), report.ReceivedBytes)
	expectedRequstSize := max(logging.ByteCount(12), logging.ByteCount(b.N)) // minimum request size is 12
	assert.Equal(b, expectedRequstSize, report.SentBytes)
	b.StopTimer()
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds()/1e6*8, "Mbps")
}
