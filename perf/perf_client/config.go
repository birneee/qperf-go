package perf_client

import (
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"qperf-go/common/qlog"
	"qperf-go/perf"
)

type Config struct {
	TlsConfig       *tls.Config
	QuicConfig      *quic.Config
	OnStreamSend    func(id quic.StreamID, count logging.ByteCount)
	OnStreamReceive func(id quic.StreamID, count logging.ByteCount)
	Qlog            qlog.Writer
}

func (c *Config) Populate() *Config {
	if c == nil {
		c = &Config{}
	}
	if c.TlsConfig == nil {
		c.TlsConfig = &tls.Config{}
	}
	if c.TlsConfig.NextProtos == nil {
		c.TlsConfig.NextProtos = []string{perf.ALPN}
	}
	return c
}
