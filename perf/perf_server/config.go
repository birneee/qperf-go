package perf_server

import (
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"qperf-go/common/qlog"
	"qperf-go/perf"
)

type Config struct {
	TlsConfig  *tls.Config
	QuicConfig *quic.Config
	QlogLabel  string
	Qlog       qlog.Writer
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
	if c.QuicConfig == nil {
		c.QuicConfig = &quic.Config{}
		c.QuicConfig.Allow0RTT = true
		c.QuicConfig.EnableDatagrams = true
	}
	return c
}
