package client

import (
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"math"
	qlog2 "qperf-go/common/qlog"
	"qperf-go/perf"
	"runtime/debug"
	"time"
)

const (
	DefaultProbeTime      = MaxProbeTime
	MaxProbeTime          = time.Duration(math.MaxInt64)
	DefaultReportInterval = 1 * time.Second
	DefaultQlogTitle      = "qperf"
	DefaultDeadline       = time.Duration(math.MaxInt64)
)

func getDefaultQlogCodeVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}

type Config struct {
	TimeToFirstByteOnly   bool
	ProbeTime             time.Duration
	ReportInterval        time.Duration
	Use0RTT               bool
	LogPrefix             string
	SendInfiniteStream    bool
	ReceiveInfiniteStream bool
	SendDatagram          bool
	ReceiveDatagram       bool
	// QlogConfig only applies to the perf qlog, not to quic-go
	QlogConfig                *qlog2.Config
	RemoteAddress             string
	TlsConfig                 *tls.Config
	ReportLostPackets         bool
	ReportMaxRTT              bool
	QuicConfig                *quic.Config
	ReconnectOnTimeoutOrReset bool
	RequestLength             uint64
	ResponseLength            uint64
	RequestInterval           time.Duration
	// RequestDeadline resets the stream if the request cannot be sent due to insufficient window sizes within the deadline
	RequestDeadline time.Duration
	// ResponseDeadline resets the stream if the response is not received within the deadline
	ResponseDeadline time.Duration
	ResponseDelay    time.Duration
	NumRequests      uint64
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
	if c.QlogConfig == nil {
		c.QlogConfig = &qlog2.Config{}
	}
	if c.QlogConfig.Title == "" {
		c.QlogConfig.Title = DefaultQlogTitle
	}
	if c.QlogConfig.VantagePoint == 0 {
		c.QlogConfig.VantagePoint = logging.PerspectiveClient
	}
	if c.QlogConfig.CodeVersion == "" {
		c.QlogConfig.CodeVersion = getDefaultQlogCodeVersion()
	}
	c.QlogConfig.Populate()
	if c.QuicConfig == nil {
		c.QuicConfig = &quic.Config{}
		c.QuicConfig.EnableDatagrams = true
	}
	if c.ReportInterval == 0 {
		c.ReportInterval = time.Duration(math.MaxInt64)
	}
	if c.NumRequests == 0 {
		if c.RequestInterval == 0 {
			c.NumRequests = 1
		} else {
			c.NumRequests = math.MaxUint64
		}
	}
	if c.ProbeTime == 0 {
		c.ProbeTime = time.Duration(math.MaxInt64)
	}
	if c.RequestDeadline == 0 {
		c.RequestDeadline = DefaultDeadline
	}
	if c.ResponseDeadline == 0 {
		c.ResponseDeadline = DefaultDeadline
	}
	return c
}
