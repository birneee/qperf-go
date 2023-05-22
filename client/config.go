package client

import (
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"qperf-go/common"
	qlog2 "qperf-go/common/qlog"
	"runtime/debug"
	"time"
)

const (
	DefaultProxyAfter     = 0 * time.Millisecond
	DefaultProbeTime      = 10 * time.Second
	DefaultReportInterval = 1 * time.Second
	DefaultQlogTitle      = "qperf"
)

func getDefaultQlogCodeVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}

type Config struct {
	TimeToFirstByteOnly bool
	ProbeTime           time.Duration
	ReportInterval      time.Duration
	Use0RTT             bool
	LogPrefix           string
	SendStream          bool
	ReceiveStream       bool
	SendDatagram        bool
	ReceiveDatagram     bool
	// output path of qlog file. {odcid} is substituted.
	QlogPathTemplate  string
	QlogConfig        *qlog2.Config
	RemoteAddresses   []string
	TlsConfig         *tls.Config
	ReportLostPackets bool
	ReportMaxRTT      bool
	QuicConfig        *quic.Config
	// if 0, address will not be updated
	NextRemoteAddrAfter time.Duration
}

func (c *Config) Populate() *Config {
	if c == nil {
		c = &Config{}
	}
	if c.TlsConfig == nil {
		c.TlsConfig = &tls.Config{}
	}
	if c.TlsConfig.NextProtos == nil {
		c.TlsConfig.NextProtos = []string{common.QperfALPN}
	}
	if c.QlogConfig == nil {
		c.QlogConfig = &qlog2.Config{}
		c.QlogConfig.VantagePoint = logging.PerspectiveClient
	}
	if c.QlogConfig.Title == "" {
		c.QlogConfig.Title = DefaultQlogTitle
	}
	if c.QlogConfig.CodeVersion == "" {
		c.QlogConfig.CodeVersion = getDefaultQlogCodeVersion()
	}
	c.QlogConfig.Populate()
	if c.QuicConfig == nil {
		c.QuicConfig = &quic.Config{}
		c.QuicConfig.EnableDatagrams = true
	}
	return c
}
