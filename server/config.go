package server

import (
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"net"
	"qperf-go/common"
	qlog2 "qperf-go/common/qlog"
	"qperf-go/perf/perf_server"
	"runtime/debug"
	"time"
)

const (
	DefaultQlogTitle = "qperf"
)

func getDefaultQlogCodeVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}

type Config struct {
	// output path of qlog file. {odcid} is substituted.
	QlogConfig            *qlog2.Config
	PerfConfig            *perf_server.Config
	Use0RTTStateRequest   bool
	ConnectionIDGenerator quic.ConnectionIDGenerator
	RouterKey             *[32]byte
	// Used instead of local socket IP.
	// Useful when listening on multiple network interfaces.
	ServerID          *net.UDPAddr
	SessionTicketKey  *[32]byte
	AddressTokenKey   *quic.TokenGeneratorKey
	StatelessResetKey *quic.StatelessResetKey
	PileInterval      time.Duration
	PileDuration      time.Duration
	Events            []common.Event
}

func (c *Config) Populate() *Config {
	if c == nil {
		c = &Config{}
	}
	if c.QlogConfig == nil {
		c.QlogConfig = &qlog2.Config{}
	}
	if c.QlogConfig.Title == "" {
		c.QlogConfig.Title = DefaultQlogTitle
	}
	if c.QlogConfig.VantagePoint == 0 {
		c.QlogConfig.VantagePoint = logging.PerspectiveServer
	}
	if c.QlogConfig.CodeVersion == "" {
		c.QlogConfig.CodeVersion = getDefaultQlogCodeVersion()
	}
	c.QlogConfig.Populate()
	c.PerfConfig = c.PerfConfig.Populate()
	if c.SessionTicketKey != nil {
		c.PerfConfig.TlsConfig.SetSessionTicketKeys([][32]byte{*c.SessionTicketKey})
	}
	return c
}
