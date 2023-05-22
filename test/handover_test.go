package test

import (
	"context"
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"github.com/stretchr/testify/require"
	"qperf-go/client"
	"qperf-go/common"
	"qperf-go/common/qlog"
	"qperf-go/server"
	"testing"
	"time"
)

func TestServerHandover(t *testing.T) {
	qlogConfig := &qlog.Config{
		ExcludeEventsByDefault: true,
	}
	//qlogConfig.SetIncludedEvents(map[string]bool{
	//	"qperf:report":           true,
	//	"qperf:total":            true,
	//	"transport:path_updated": true,
	//})
	server1Counter := SentStreamDataCountTracer{}
	server1 := server.Listen("localhost:0", "server1", (&server.Config{
		TlsConfig: &tls.Config{
			Certificates: []tls.Certificate{common.GenerateCert()},
		},
		QlogConfig: qlogConfig,
		ServeState: true,
		QuicConfig: &quic.Config{
			Tracer: func(ctx context.Context, perspective logging.Perspective, id quic.ConnectionID) logging.ConnectionTracer {
				return &server1Counter
			},
		},
	}).Populate())
	server2Counter := SentStreamDataCountTracer{}
	server2 := server.Listen("localhost:0", "server2", (&server.Config{
		QlogConfig:  qlogConfig,
		StateServer: server1.Addr(),
		StateTransferConfig: &quic.StateTransferConfig{
			TlsConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		QuicConfig: &quic.Config{
			Tracer: func(ctx context.Context, perspective logging.Perspective, id quic.ConnectionID) logging.ConnectionTracer {
				return &server2Counter
			},
		},
	}).Populate())
	client := client.Dial((&client.Config{
		ReportInterval:  time.Second / 10,
		QlogConfig:      qlogConfig,
		ProbeTime:       time.Second,
		ReceiveStream:   true,
		RemoteAddresses: []string{server1.Addr().String(), server2.Addr().String()},
		TlsConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}).Populate())

	<-time.After(time.Second / 2)
	client.UseNextRemoteAddr()

	<-client.Context().Done()
	server1.Close(nil)
	server2.Close(nil)
	require.Greater(t, int(server1Counter.Count), 1000)
	require.Greater(t, int(server2Counter.Count), 1000)
}
