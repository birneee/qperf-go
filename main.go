package main

import (
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"github.com/urfave/cli/v2"
	"net"
	"os"
	"qperf-go/client"
	"qperf-go/common"
	"qperf-go/proxy"
	"qperf-go/server"
	"time"
)

const defaultProxyControlPort = 18081
const defaultProxyTLSCertificateFile = "proxy.crt"
const defaultProxyTLSKeyFile = "proxy.key"

func main() {
	app := &cli.App{
		Name:  "qperf-go",
		Usage: "A performance measurement tool for QUIC similar to iperf",
		Commands: []*cli.Command{
			{
				Name:  "proxy",
				Usage: "run in proxy mode",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "addr",
						Usage: "address of the proxy to listen on",
						Value: "0.0.0.0",
					},
					&cli.UintFlag{
						Name:  "port",
						Usage: "port of the proxy to listen on, for control connections",
						Value: defaultProxyControlPort,
					},
					&cli.StringFlag{
						Name:  "tls-cert",
						Usage: "certificate file to use",
						Value: defaultProxyTLSCertificateFile,
					},
					&cli.StringFlag{
						Name:  "tls-key",
						Usage: "key file to use",
						Value: defaultProxyTLSKeyFile,
					},
					&cli.StringFlag{
						Name:  "next-proxy",
						Usage: "the additional, server-facing proxy to use, in the form \"host:port\", default port 18081 if not specified",
					},
					&cli.StringFlag{
						Name:  "next-proxy-cert",
						Usage: "certificate file to trust the next proxy",
						Value: "proxy.crt",
					},
					&cli.UintFlag{
						Name:  "client-facing-initial-congestion-window",
						Usage: "the initial congestion window to use on client facing proxy connections, in number of packets",
						Value: quic.DefaultInitialCongestionWindow,
					},
					&cli.UintFlag{
						Name:  "client-facing-min-congestion-window",
						Usage: "the minimum congestion window to use on client facing proxy connections, in number of packets",
						Value: quic.DefaultMinCongestionWindow,
					},
					&cli.UintFlag{
						Name:  "client-facing-max-congestion-window",
						Usage: "the maximum congestion window to use on client facing proxy connections, in number of packets",
						Value: quic.DefaultMaxCongestionWindow,
					},
					&cli.StringFlag{
						Name:  "client-facing-initial-receive-window",
						Usage: "the initial receive window on the client facing proxy connection, in bytes, overwrites the value from the handover state",
					},
					&cli.StringFlag{
						Name:  "server-facing-initial-receive-window",
						Usage: "the initial receive window on the server facing proxy connection, in bytes, overwrites the value from the handover state",
					},
					&cli.StringFlag{
						Name:  "server-facing-max-receive-window",
						Usage: "the maximum receive window on the server facing proxy connection, in bytes, overwrites the value from the handover state",
					},
					&cli.BoolFlag{
						Name:  "0rtt",
						Usage: "gather 0-RTT information to the next proxy beforehand",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "qlog",
						Usage: "create qlog file",
					},
					&cli.StringFlag{
						Name:  "qlog-prefix",
						Usage: "the prefix of the qlog file name",
						Value: "proxy",
					},
					&cli.StringFlag{
						Name:  "log-prefix",
						Usage: "the prefix of the command line output",
						Value: "",
					},
				},
				Action: func(c *cli.Context) error {
					var nextProxyAddr *net.UDPAddr
					if c.IsSet("next-proxy") {
						var err error
						nextProxyAddr, err = common.ParseResolveHost(c.String("next-proxy"), common.DefaultProxyControlPort)
						if err != nil {
							panic(err)
						}
					}
					var clientSideInitialReceiveWindow uint64
					if c.IsSet("client-facing-initial-receive-window") {
						var err error
						clientSideInitialReceiveWindow, err = common.ParseByteCountWithUnit(c.String("client-facing-initial-receive-window"))
						if err != nil {
							return fmt.Errorf("failed to parse client-facing-initial-receive-window: %w", err)
						}
					}
					var serverSideInitialReceiveWindow uint64
					if c.IsSet("server-facing-initial-receive-window") {
						var err error
						serverSideInitialReceiveWindow, err = common.ParseByteCountWithUnit(c.String("server-facing-initial-receive-window"))
						if err != nil {
							return fmt.Errorf("failed to parse server-facing-initial-receive-window: %w", err)
						}
					}
					var serverSideMaxReceiveWindow uint64
					if c.IsSet("server-facing-max-receive-window") {
						var err error
						serverSideMaxReceiveWindow, err = common.ParseByteCountWithUnit(c.String("server-facing-max-receive-window"))
						if err != nil {
							return fmt.Errorf("failed to parse server-facing-max-receive-window: %w", err)
						}
					}
					proxy.Run(
						net.UDPAddr{
							IP:   net.ParseIP(c.String("addr")),
							Port: c.Int("port"),
						},
						c.String("tls-cert"),
						c.String("tls-key"),
						nextProxyAddr,
						c.String("next-proxy-cert"),
						uint32(c.Uint("client-facing-initial-congestion-window")),
						uint32(c.Uint("client-facing-min-congestion-window")),
						uint32(c.Uint("client-facing-max-congestion-window")),
						clientSideInitialReceiveWindow,
						serverSideInitialReceiveWindow,
						serverSideMaxReceiveWindow,
						c.Bool("0rtt"),
						c.Bool("qlog"),
						c.String("log-prefix"),
						c.String("qlog-prefix"),
					)
					return nil
				},
			},
			{
				Name:  "client",
				Usage: "run in client mode",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "addr",
						Usage:    "address to connect to, in the form \"host:port\", default port 18080 if not specified",
						Required: true,
					},
					&cli.BoolFlag{
						Name:  "ttfb",
						Usage: "measure time for connection establishment and first byte only",
					},
					&cli.BoolFlag{
						Name:  "print-raw",
						Usage: "output raw statistics, don't calculate metric prefixes",
					},
					&cli.BoolFlag{
						Name:  "qlog",
						Usage: "create qlog file",
					},
					&cli.UintFlag{
						Name:  "migrate",
						Usage: "seconds after which the udp socket is migrated",
					},
					&cli.StringFlag{
						Name:  "proxy",
						Usage: "the proxy to use, in the form \"host:port\", default port 18081 if not specified",
					},
					&cli.UintFlag{
						Name:  "t",
						Usage: "run for this many seconds",
						Value: 10,
					},
					&cli.Float64Flag{
						Name:    "report-interval",
						Aliases: []string{"i"},
						Usage:   "seconds between each statistics report",
						Value:   1.0,
					},
					&cli.StringFlag{
						Name:  "tls-cert",
						Usage: "certificate file to trust the server",
						Value: "server.crt",
					},
					&cli.StringFlag{
						Name:  "tls-proxy-cert",
						Usage: "certificate file to trust the proxy",
						Value: "proxy.crt",
					},
					&cli.UintFlag{
						Name:  "initial-congestion-window",
						Usage: "the initial congestion window to use, in number of packets",
						Value: quic.DefaultInitialCongestionWindow,
					},
					&cli.StringFlag{
						Name:  "initial-receive-window",
						Usage: "the initial stream-level receive window, in bytes (the connection-level window is 1.5 times higher)",
						Value: "512KiB",
					},
					&cli.StringFlag{
						Name:  "max-receive-window",
						Usage: "the maximum stream-level receive window, in bytes (the connection-level window is 1.5 times higher)",
						Value: "6MiB",
					},
					&cli.BoolFlag{
						Name:  "0rtt",
						Usage: "gather 0-RTT information to the server beforehand",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "proxy-0rtt",
						Usage: "gather 0-RTT information to the proxy beforehand",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "xse",
						Usage: "use XSE-QUIC extension; handshake will fail if not supported by server",
						Value: false,
					},
					&cli.StringFlag{
						Name:  "qlog-prefix",
						Usage: "the prefix of the qlog file name",
						Value: "client",
					},
					&cli.StringFlag{
						Name:  "log-prefix",
						Usage: "the prefix of the command line output",
						Value: "",
					},
				},
				Action: func(c *cli.Context) error {
					var proxyAddr *net.UDPAddr
					if c.IsSet("proxy") {
						var err error
						proxyAddr, err = common.ParseResolveHost(c.String("proxy"), common.DefaultProxyControlPort)
						if err != nil {
							panic(err)
						}
					}
					serverAddr, err := common.ParseResolveHost(c.String("addr"), common.DefaultQperfServerPort)
					if err != nil {
						println("invalid server address")
						panic(err)
					}
					initialReceiveWindow, err := common.ParseByteCountWithUnit(c.String("initial-receive-window"))
					if err != nil {
						return fmt.Errorf("failed to parse receive-window: %w", err)
					}
					maxReceiveWindow, err := common.ParseByteCountWithUnit(c.String("max-receive-window"))
					if err != nil {
						return fmt.Errorf("failed to parse receive-window: %w", err)
					}
					client.Run(
						*serverAddr,
						c.Bool("ttfb"),
						c.Bool("print-raw"),
						c.Bool("qlog"),
						time.Duration(c.Uint64("migrate"))*time.Second,
						proxyAddr,
						time.Duration(c.Uint("t"))*time.Second,
						time.Duration(c.Float64("report-interval")*float64(time.Second)),
						c.String("tls-cert"),
						c.String("tls-proxy-cert"),
						uint32(c.Uint("initial-congestion-window")),
						initialReceiveWindow,
						maxReceiveWindow,
						c.Bool("0rtt"),
						c.Bool("proxy-0rtt"),
						c.Bool("xse"),
						c.String("log-prefix"),
						c.String("qlog-prefix"),
					)
					return nil
				},
			},
			{
				Name:  "server",
				Usage: "run in server mode",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "addr",
						Usage: "address to listen on",
						Value: "0.0.0.0",
					},
					&cli.UintFlag{
						Name:  "port",
						Usage: "port to listen on",
						Value: common.DefaultQperfServerPort,
					},
					&cli.BoolFlag{
						Name:  "qlog",
						Usage: "create qlog file",
					},
					&cli.UintFlag{
						Name:  "migrate",
						Usage: "seconds after which the udp socket is migrated",
					},
					&cli.StringFlag{
						Name:  "tls-cert",
						Usage: "certificate file to use",
						Value: "server.crt",
					},
					&cli.StringFlag{
						Name:  "tls-key",
						Usage: "key file to use",
						Value: "server.key",
					},
					&cli.UintFlag{
						Name:  "initial-congestion-window",
						Usage: "the initial congestion window to use, in number of packets",
						Value: quic.DefaultInitialCongestionWindow,
					},
					&cli.UintFlag{
						Name:  "min-congestion-window",
						Usage: "the minimum congestion window to use, in number of packets",
						Value: quic.DefaultMinCongestionWindow,
					},
					&cli.UintFlag{
						Name:  "max-congestion-window",
						Usage: "the maximum congestion window to use, in number of packets",
						Value: quic.DefaultMaxCongestionWindow,
					},
					&cli.StringFlag{
						Name:  "initial-receive-window",
						Usage: "the initial stream-level receive window, in bytes (the connection-level window is 1.5 times higher)",
						Value: "512KiB",
					},
					&cli.StringFlag{
						Name:  "max-receive-window",
						Usage: "the maximum stream-level receive window, in bytes (the connection-level window is 1.5 times higher)",
						Value: "6MiB",
					},
					&cli.BoolFlag{
						Name:  "no-xse",
						Usage: "disable XSE-QUIC extension; XSE-QUIC handshakes will fail",
						Value: false,
					},
					&cli.StringFlag{
						Name:  "qlog-prefix",
						Usage: "the prefix of the qlog file name",
						Value: "server",
					},
					&cli.StringFlag{
						Name:  "log-prefix",
						Usage: "the prefix of the command line output",
						Value: "",
					},
				},
				Action: func(c *cli.Context) error {
					initialReceiveWindow, err := common.ParseByteCountWithUnit(c.String("initial-receive-window"))
					if err != nil {
						return fmt.Errorf("failed to parse receive-window: %w", err)
					}
					maxReceiveWindow, err := common.ParseByteCountWithUnit(c.String("max-receive-window"))
					if err != nil {
						return fmt.Errorf("failed to parse receive-window: %w", err)
					}
					server.Run(net.UDPAddr{
						IP:   net.ParseIP(c.String("addr")),
						Port: c.Int("port"),
					},
						c.Bool("qlog"),
						time.Duration(c.Uint64("migrate"))*time.Second,
						c.String("tls-cert"),
						c.String("tls-key"),
						uint32(c.Uint("initial-congestion-window")),
						uint32(c.Uint("min-congestion-window")),
						uint32(c.Uint("max-congestion-window")),
						initialReceiveWindow,
						maxReceiveWindow,
						c.Bool("no-xse"),
						c.String("log-prefix"),
						c.String("qlog-prefix"),
					)
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
