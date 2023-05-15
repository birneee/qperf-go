package main

import (
	"crypto/tls"
	"fmt"
	"github.com/urfave/cli/v2"
	"net"
	"os"
	"qperf-go/client"
	"qperf-go/common"
	"qperf-go/server"
	"runtime/pprof"
	"time"
)

func clientCommand(config *client.Config) *cli.Command {
	return &cli.Command{
		Name:  "client",
		Usage: "run in client mode",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "remote-addr",
				Aliases:  []string{"a"},
				Usage:    fmt.Sprintf("address to connect to, in the form \"host:port\", default port %d if not specified.", common.DefaultQperfServerPort),
				Required: true,
				Action: func(ctx *cli.Context, s string) error {
					config.RemoteAddress = common.AppendPortIfNotSpecified(s, common.DefaultQperfServerPort)
					return nil
				},
			},
			&cli.BoolFlag{
				Name:  "ttfb",
				Usage: "measure time for connection establishment and first byte only",
			},
			&cli.BoolFlag{
				Name:  "print-raw",
				Usage: "output raw statistics, don't calculate metric prefixes",
			},
			&cli.StringFlag{
				Name:  "qlog",
				Usage: "output path of qlog file. {odcid} is automatically substituted.",
				Action: func(context *cli.Context, s string) error {
					config.QlogPathTemplate = s
					return nil
				},
			},
			&cli.StringFlag{
				Name: "qlog-level",
				Aliases: []string{
					"ql",
				},
				Usage:      "verbosity of qlog output. e.g. info, full",
				Value:      "info",
				HasBeenSet: true,
				Action: func(context *cli.Context, s string) error {
					switch s {
					case "info":
						config.QlogConfig.ExcludeEventsByDefault = true
						config.QlogConfig.SetIncludedEvents(common.QlogLevelInfoEvents)
					case "full":
						config.QlogConfig.ExcludeEventsByDefault = false
					default:
						return fmt.Errorf("unsupported qlog-level: %s", s)
					}
					return nil
				},
			},
			&cli.Float64Flag{
				Name:       "t",
				Usage:      "run for this many seconds",
				Value:      client.DefaultProbeTime.Seconds(),
				HasBeenSet: true,
				Action: func(context *cli.Context, f float64) error {
					config.ProbeTime = time.Duration(f * float64(time.Second))
					return nil
				},
			},
			&cli.Float64Flag{
				Name:    "report-interval",
				Aliases: []string{"i"},
				Usage:   "seconds between each statistics report",
				Value:   client.DefaultReportInterval.Seconds(),
			},
			&cli.StringSliceFlag{
				Name:  "cert-pool",
				Usage: "certificate files to trust",
				Action: func(context *cli.Context, paths []string) error {
					config.TlsConfig.RootCAs = common.NewCertPoolFromFiles(paths...)
					return nil
				},
			},
			&cli.StringFlag{
				Name:       "initial-receive-window",
				Usage:      "the initial stream-level receive window, in bytes (the connection-level window is 1.5 times higher)",
				Value:      "512KiB",
				HasBeenSet: true,
				Action: func(context *cli.Context, s string) error {
					var err error
					config.QuicConfig.InitialStreamReceiveWindow, err = common.ParseByteCountWithUnit(s)
					if err != nil {
						return fmt.Errorf("failed to parse receive-window: %w", err)
					}
					config.QuicConfig.InitialConnectionReceiveWindow = uint64(float64(config.QuicConfig.InitialStreamReceiveWindow) * common.ConnectionFlowControlMultiplier)
					return nil
				},
			},
			&cli.StringFlag{
				Name:       "max-receive-window",
				Usage:      "the maximum stream-level receive window, in bytes (the connection-level window is 1.5 times higher)",
				Value:      "6MiB",
				HasBeenSet: true,
				Action: func(context *cli.Context, s string) error {
					var err error
					config.QuicConfig.MaxStreamReceiveWindow, err = common.ParseByteCountWithUnit(s)
					if err != nil {
						return fmt.Errorf("failed to parse max-receive-window: %w", err)
					}
					config.QuicConfig.MaxConnectionReceiveWindow = uint64(float64(config.QuicConfig.MaxStreamReceiveWindow) * common.ConnectionFlowControlMultiplier)
					return nil
				},
			},
			&cli.BoolFlag{
				Name:  "0rtt",
				Usage: "gather 0-RTT information to the server beforehand",
				Value: false,
				Action: func(context *cli.Context, b bool) error {
					config.Use0RTT = b
					return nil
				},
			},
			&cli.StringFlag{
				Name:  "log-prefix",
				Usage: "the prefix of the command line output",
				Value: "",
			},
			&cli.BoolFlag{
				Name:  "receive-stream",
				Usage: "stream data from server. Disable by --receive=0",
				Value: false,
				Action: func(context *cli.Context, b bool) error {
					config.ReceiveStream = b
					return nil
				},
			},
			&cli.BoolFlag{
				Name:  "send-stream",
				Usage: "stream data to server",
				Value: false,
				Action: func(context *cli.Context, b bool) error {
					config.SendStream = b
					return nil
				},
			},
			&cli.BoolFlag{
				Name:  "send-datagram",
				Usage: "send datagrams to server",
				Value: false,
				Action: func(context *cli.Context, b bool) error {
					config.SendDatagram = b
					return nil
				},
			},
			&cli.BoolFlag{
				Name:  "receive-datagram",
				Usage: "receive datagrams to server",
				Value: false,
				Action: func(context *cli.Context, b bool) error {
					config.ReceiveDatagram = b
					return nil
				},
			},
			&cli.BoolFlag{
				Name: "tls-skip-verify",
				Aliases: []string{
					"tsv",
				},
				Usage: "skip verification ot the server's certificate chain and host name",
				Value: false,
				Action: func(context *cli.Context, b bool) error {
					config.TlsConfig.InsecureSkipVerify = b
					return nil
				},
			},
			&cli.BoolFlag{
				Name: "packet-loss",
				Aliases: []string{
					"pl",
				},
				Usage: "include number of lost packets in the reports",
				Value: false,
				Action: func(context *cli.Context, b bool) error {
					config.ReportLostPackets = b
					return nil
				},
			},
			&cli.BoolFlag{
				Name: "max-rtt",
				Aliases: []string{
					"xr",
				},
				Usage: "include the maximum RTT in the reports",
				Value: false,
				Action: func(context *cli.Context, b bool) error {
					config.ReportMaxRTT = b
					return nil
				},
			},
		},
		Action: func(c *cli.Context) error {
			if !config.ReceiveStream && !config.SendStream && !config.ReceiveDatagram && !config.SendDatagram {
				config.ReceiveStream = true // receive stream if nothing else is specified
			}

			config.QuicConfig.MaxStreamReceiveWindow = common.Max(config.QuicConfig.InitialStreamReceiveWindow, config.QuicConfig.MaxStreamReceiveWindow)

			config.QuicConfig.MaxConnectionReceiveWindow = common.Max(config.QuicConfig.InitialConnectionReceiveWindow, config.QuicConfig.MaxConnectionReceiveWindow)

			config.TimeToFirstByteOnly = c.Bool("ttfb")
			config.ReportInterval = time.Duration(c.Float64("report-interval") * float64(time.Second))
			config.LogPrefix = c.String("log-prefix")
			client.Run(config)
			return nil
		},
	}
}

func serverCommand(config *server.Config) *cli.Command {
	return &cli.Command{
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
			&cli.StringFlag{
				Name:  "qlog",
				Usage: "output path of qlog file. {odcid} is automatically substituted.",
				Action: func(context *cli.Context, s string) error {
					config.QlogPathTemplate = s
					return nil
				},
			},
			&cli.StringFlag{
				Name:       "qlog-level",
				Usage:      "verbosity of qlog output. e.g. info, full",
				Value:      "info",
				HasBeenSet: true,
				Action: func(context *cli.Context, s string) error {
					switch s {
					case "info":
						config.QlogConfig.ExcludeEventsByDefault = true
						config.QlogConfig.SetIncludedEvents(common.QlogLevelInfoEvents)
					case "full":
						config.QlogConfig.ExcludeEventsByDefault = false
					default:
						return fmt.Errorf("unsupported qlog-level: %s", s)
					}
					return nil
				},
			},
			&cli.StringFlag{
				Name:  "tls-cert",
				Usage: "certificate file to use",
			},
			&cli.StringFlag{
				Name:  "tls-key",
				Usage: "key file to use",
				Action: func(ctx *cli.Context, s string) error {
					if !ctx.IsSet("tls-cert") {
						return fmt.Errorf("-tls-cert must also be set")
					}
					cert, err := tls.LoadX509KeyPair(ctx.String("tls-cert"), s)
					if err != nil {
						return err
					}
					config.TlsConfig.Certificates = []tls.Certificate{cert}
					return nil
				},
			},
			&cli.StringFlag{
				Name:       "initial-receive-window",
				Usage:      "the initial stream-level receive window, in bytes (the connection-level window is 1.5 times higher)",
				Value:      "512KiB",
				HasBeenSet: true,
				Action: func(context *cli.Context, s string) error {
					var err error
					config.QuicConfig.InitialStreamReceiveWindow, err = common.ParseByteCountWithUnit(s)
					if err != nil {
						return fmt.Errorf("failed to parse receive-window: %w", err)
					}
					config.QuicConfig.InitialConnectionReceiveWindow = uint64(float64(config.QuicConfig.InitialStreamReceiveWindow) * common.ConnectionFlowControlMultiplier)
					return nil
				},
			},
			&cli.StringFlag{
				Name:       "max-receive-window",
				Usage:      "the maximum stream-level receive window, in bytes (the connection-level window is 1.5 times higher)",
				Value:      "6MiB",
				HasBeenSet: true,
				Action: func(context *cli.Context, s string) error {
					var err error
					config.QuicConfig.MaxStreamReceiveWindow, err = common.ParseByteCountWithUnit(s)
					if err != nil {
						return fmt.Errorf("failed to parse max-receive-window: %w", err)
					}
					config.QuicConfig.MaxConnectionReceiveWindow = uint64(float64(config.QuicConfig.MaxStreamReceiveWindow) * common.ConnectionFlowControlMultiplier)
					return nil
				},
			},
			&cli.StringFlag{
				Name:  "log-prefix",
				Usage: "the prefix of the command line output",
				Value: "",
			},
		},
		Action: func(c *cli.Context) error {
			if config.TlsConfig.Certificates == nil {
				fmt.Printf("generate self signed TLS certificate\n")
				config.TlsConfig.Certificates = []tls.Certificate{common.GenerateCert()}
			}

			config.QuicConfig.MaxStreamReceiveWindow = common.Max(config.QuicConfig.InitialStreamReceiveWindow, config.QuicConfig.MaxStreamReceiveWindow)

			config.QuicConfig.MaxConnectionReceiveWindow = common.Max(config.QuicConfig.InitialConnectionReceiveWindow, config.QuicConfig.MaxConnectionReceiveWindow)

			server.Run(net.UDPAddr{
				IP:   net.ParseIP(c.String("addr")),
				Port: c.Int("port"),
			},
				c.String("log-prefix"),
				config,
			)
			return nil
		},
	}
}

func main() {
	clientConfig := (&client.Config{}).Populate()
	serverConfig := (&server.Config{}).Populate()

	var doOnStop []func()

	app := &cli.App{
		Name:  "qperf-go",
		Usage: "A performance measurement tool for QUIC similar to iperf",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "cpu-profile",
				Usage: "output path of prof file",
				Action: func(context *cli.Context, fileName string) error {
					w, err := os.Create(fileName)
					if err != nil {
						return err
					}
					err = pprof.StartCPUProfile(w)
					if err != nil {
						return err
					}
					doOnStop = append(doOnStop, func() {
						pprof.StopCPUProfile()
						_ = w.Close()
					})
					return nil
				},
			},
		},
		Commands: []*cli.Command{
			clientCommand(clientConfig),
			serverCommand(serverConfig),
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
	for _, d := range doOnStop {
		d()
	}
}
