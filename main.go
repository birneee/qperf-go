package main

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/quic-go/quic-go"
	qlog2 "github.com/quic-go/quic-go/qlog"
	"github.com/urfave/cli/v2"
	"net"
	"os"
	"qperf-go/client"
	"qperf-go/common"
	"qperf-go/common/qlog"
	"qperf-go/perf"
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
				Usage:    fmt.Sprintf("address to connect to, in the form \"host:port\", default port %d if not specified.", perf.DefaultServerPort),
				Required: true,
				Action: func(ctx *cli.Context, s string) error {
					config.RemoteAddress = common.AppendPortIfNotSpecified(s, perf.DefaultServerPort)
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
			&cli.UintFlag{
				Name:  "qlog-queue",
				Usage: "set size of the qlog event in-memory queue",
				Value: qlog.DefaultMemoryQueueSize,
				Action: func(context *cli.Context, i uint) error {
					config.QlogConfig.MemoryQueueSize = int(i)
					return nil
				},
			},
			&cli.DurationFlag{
				Name:        "time",
				Aliases:     []string{"t"},
				Usage:       "run for this long",
				Value:       client.DefaultProbeTime,
				Destination: &config.ProbeTime,
			},
			&cli.DurationFlag{
				Name:        "report-interval",
				Aliases:     []string{"i"},
				Usage:       "seconds between each statistics report",
				Value:       client.DefaultReportInterval,
				Destination: &config.ReportInterval,
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
				Value:      "768KiB",
				HasBeenSet: true,
				Action: func(context *cli.Context, s string) error {
					win, err := common.ParseByteCountWithUnit(s)
					if err != nil {
						return fmt.Errorf("failed to parse receive-window: %w", err)
					}
					config.QuicConfig.InitialStreamReceiveWindow = win
					config.QuicConfig.InitialConnectionReceiveWindow = win
					return nil
				},
			},
			&cli.StringFlag{
				Name:       "max-receive-window",
				Usage:      "the maximum stream-level receive window, in bytes (the connection-level window is 1.5 times higher)",
				Value:      "9MiB",
				HasBeenSet: true,
				Action: func(context *cli.Context, s string) error {
					win, err := common.ParseByteCountWithUnit(s)
					if err != nil {
						return fmt.Errorf("failed to parse max-receive-window: %w", err)
					}
					config.QuicConfig.MaxStreamReceiveWindow = win
					config.QuicConfig.MaxConnectionReceiveWindow = win
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
			&cli.BoolFlag{
				Name:  "receive-stream",
				Usage: "stream data from server. Disable by --receive-stream=0",
				Value: false,
				Action: func(ctx *cli.Context, b bool) error {
					if ctx.IsSet("response-length") {
						return fmt.Errorf("either set receive-stream or response-length")
					}
					config.ReceiveInfiniteStream = b
					return nil
				},
			},
			&cli.BoolFlag{
				Name:  "send-stream",
				Usage: "stream data to server. Disable by --send-stream=0",
				Value: false,
				Action: func(ctx *cli.Context, b bool) error {
					if ctx.IsSet("request-length") {
						return fmt.Errorf("either set send-stream or request-length")
					}
					config.SendInfiniteStream = b
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
			&cli.BoolFlag{
				Name: "reconnect",
				Aliases: []string{
					"r",
				},
				Usage: "try reconnecting to server on QUIC idle timeout or stateless reset",
				Value: false,
				Action: func(context *cli.Context, b bool) error {
					config.ReconnectOnTimeoutOrReset = b
					return nil
				},
			},
			&cli.BoolFlag{
				Name:  "min-timeout",
				Usage: "use the minimum idle timeout of 3 PTOs (RFC 9000 10.1)",
				Value: false,
				Action: func(context *cli.Context, b bool) error {
					if b {
						config.QuicConfig.MaxIdleTimeout = time.Nanosecond
					}
					return nil
				},
			},
			&cli.Uint64Flag{
				Name:  "request-length",
				Usage: "bytes sent per stream request",
				Value: 0,
				Action: func(context *cli.Context, v uint64) error {
					config.RequestLength = v
					return nil
				},
			},
			&cli.DurationFlag{
				Name:        "request-interval",
				Usage:       "time after which a new request is sent",
				Value:       0,
				Destination: &config.RequestInterval,
			},
			&cli.Uint64Flag{
				Name: "request-number",
				Aliases: []string{
					"n",
				},
				Usage:      "the number of requests to send",
				HasBeenSet: true,
				Action: func(context *cli.Context, i uint64) error {
					config.NumRequests = i
					return nil
				},
			},
			&cli.Uint64Flag{
				Name:  "response-length",
				Usage: "bytes received per stream response",
				Value: 0,
				Action: func(context *cli.Context, v uint64) error {
					config.ResponseLength = v
					return nil
				},
			},
			&cli.DurationFlag{
				Name:        "response-delay",
				Usage:       "time that the server waits until responding to received requests",
				Value:       0,
				Destination: &config.ResponseDelay,
			},
			&cli.DurationFlag{
				Name:        "response-deadline",
				Usage:       "time after which to cancel sending the request and receiving its response",
				Value:       client.DefaultDeadline,
				Destination: &config.ResponseDeadline,
			},
			&cli.DurationFlag{
				Name:        "request-deadline",
				Usage:       "time after which to cancel sending the request",
				Value:       client.DefaultDeadline,
				Destination: &config.RequestDeadline,
			},
			&cli.BoolFlag{
				Name:  "gso",
				Usage: "enable generic segmentation offload",
				Value: false,
				Action: func(context *cli.Context, b bool) error {
					var err error
					if b {
						err = os.Setenv("QUIC_GO_ENABLE_GSO", "1")

					} else {
						err = os.Unsetenv("QUIC_GO_ENABLE_GSO")
					}
					if err != nil {
						return err
					}
					return nil
				},
			},
		},
		Action: func(c *cli.Context) error {
			if !config.ReceiveInfiniteStream &&
				!config.SendInfiniteStream &&
				!config.ReceiveDatagram &&
				!config.SendDatagram &&
				config.RequestLength == 0 &&
				config.ResponseLength == 0 {
				config.ReceiveInfiniteStream = true // receive stream if nothing else is specified
			}

			if config.ProbeTime == 0 {
				if config.ReceiveInfiniteStream ||
					config.SendInfiniteStream ||
					config.ReceiveDatagram ||
					config.SendDatagram ||
					(config.RequestInterval != 0 && config.NumRequests == 0) {
					config.ProbeTime = client.DefaultProbeTime
				} else {
					config.ProbeTime = client.MaxProbeTime // stop after transaction not after time
				}
			}

			config.QuicConfig.MaxStreamReceiveWindow = common.Max(config.QuicConfig.InitialStreamReceiveWindow, config.QuicConfig.MaxStreamReceiveWindow)

			config.QuicConfig.MaxConnectionReceiveWindow = common.Max(config.QuicConfig.InitialConnectionReceiveWindow, config.QuicConfig.MaxConnectionReceiveWindow)

			config.TimeToFirstByteOnly = c.Bool("ttfb")
			client := client.Dial(config)
			<-client.Context().Done()
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
				Value: perf.DefaultServerPort,
			},
			&cli.UintFlag{
				Name:  "qlog-queue",
				Usage: "set size of the qlog event in-memory queue",
				Value: qlog.DefaultMemoryQueueSize,
				Action: func(context *cli.Context, i uint) error {
					config.QlogConfig.MemoryQueueSize = int(i)
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
					config.PerfConfig.TlsConfig.Certificates = []tls.Certificate{cert}
					return nil
				},
			},
			&cli.StringFlag{
				Name:       "initial-receive-window",
				Usage:      "the initial stream-level receive window, in bytes (the connection-level window is 1.5 times higher)",
				Value:      "768KiB",
				HasBeenSet: true,
				Action: func(context *cli.Context, s string) error {
					win, err := common.ParseByteCountWithUnit(s)
					if err != nil {
						return fmt.Errorf("failed to parse receive-window: %w", err)
					}
					config.PerfConfig.QuicConfig.InitialStreamReceiveWindow = win
					config.PerfConfig.QuicConfig.InitialConnectionReceiveWindow = win
					return nil
				},
			},
			&cli.StringFlag{
				Name:       "max-receive-window",
				Usage:      "the maximum stream-level receive window, in bytes (the connection-level window is 1.5 times higher)",
				Value:      "9MiB",
				HasBeenSet: true,
				Action: func(context *cli.Context, s string) error {
					win, err := common.ParseByteCountWithUnit(s)
					if err != nil {
						return fmt.Errorf("failed to parse max-receive-window: %w", err)
					}
					config.PerfConfig.QuicConfig.MaxStreamReceiveWindow = win
					config.PerfConfig.QuicConfig.MaxConnectionReceiveWindow = win
					return nil
				},
			},
			&cli.BoolFlag{
				Name:       "0rtt-state-request",
				Usage:      "use 0-rtt connection for requests to state server",
				Value:      false,
				HasBeenSet: true,
				Action: func(context *cli.Context, b bool) error {
					config.Use0RTTStateRequest = b
					return nil
				},
			},
			&cli.StringFlag{
				Name:  "session-ticket-key",
				Usage: "TLS session ticket key used for 0-RTT; value must be 32 byte and base64 encoded; if not set a random key is generated",
				Value: "",
				Action: func(ctx *cli.Context, s string) error {
					key, err := base64.StdEncoding.DecodeString(s)
					if err != nil {
						return fmt.Errorf("failed to parse session ticket key: %s", err)
					}
					if len(key) != 32 {
						return fmt.Errorf("failed to parse session ticket key: must be 32 byte")
					}
					array := ([32]byte)(key)
					config.SessionTicketKey = &array
					return nil
				},
			},
			&cli.StringFlag{
				Name:  "address-token-key",
				Usage: "QUIC address token key used for 0-RTT; value must be 32 byte and base64 encoded; if not set a random key is generated",
				Value: "",
				Action: func(ctx *cli.Context, s string) error {
					key, err := base64.StdEncoding.DecodeString(s)
					if err != nil {
						return fmt.Errorf("failed to parse session ticket key: %s", err)
					}
					if len(key) != 32 {
						return fmt.Errorf("failed to parse session ticket key: must be 32 byte")
					}
					config.AddressTokenKey = (*quic.TokenGeneratorKey)(key)
					return nil
				},
			},
			&cli.StringFlag{
				Name:  "stateless-reset-key",
				Usage: "Key used to generate stateless resets tokens; value must be 32 byte and base64 encoded; if not set stateless reset is disabled",
				Action: func(ctx *cli.Context, s string) error {
					key, err := base64.StdEncoding.DecodeString(s)
					if err != nil {
						return fmt.Errorf("failed to parse stateless reset key: %s", err)
					}
					if len(key) != 32 {
						return fmt.Errorf("failed to parse stateless reset key: must be 32 byte")
					}
					config.StatelessResetKey = (*quic.StatelessResetKey)(key)
					return nil
				},
			},
			&cli.StringFlag{
				Name:  "router-key",
				Usage: "Key used for connection id routing; value must be 32 byte and base64 encoded; if not set connection id routing is disabled",
				Action: func(ctx *cli.Context, s string) error {
					key, err := base64.StdEncoding.DecodeString(s)
					if err != nil {
						return fmt.Errorf("failed to parse stateless reset key: %s", err)
					}
					if len(key) != 32 {
						return fmt.Errorf("failed to parse stateless reset key: must be 32 byte")
					}
					config.RouterKey = (*[32]byte)(key)
					return nil
				},
			},
			&cli.StringFlag{
				Name:       "log-label",
				Value:      "server",
				HasBeenSet: true,
			},
			&cli.StringFlag{
				Name:  "server-id",
				Usage: "tbd",
				Action: func(ctx *cli.Context, s string) error {
					if !ctx.IsSet("router-key") {
						return fmt.Errorf("server-id option requires router-key option")
					}
					var err error
					s = common.AppendPortIfNotSpecified(s, perf.DefaultServerPort)
					config.ServerID, err = net.ResolveUDPAddr("udp", s)
					if err != nil {
						return err
					}
					return nil
				},
			},
			&cli.BoolFlag{
				Name:  "gso",
				Usage: "enable generic segmentation offload",
				Value: false,
				Action: func(context *cli.Context, b bool) error {
					var err error
					if b {
						err = os.Setenv("QUIC_GO_ENABLE_GSO", "1")

					} else {
						err = os.Unsetenv("QUIC_GO_ENABLE_GSO")
					}
					if err != nil {
						return err
					}
					return nil
				},
			},
			&cli.IntFlag{
				Name:  "max-incoming-streams",
				Usage: "maximum allowed number of incoming streams",
				Action: func(ctx *cli.Context, i int) error {
					config.PerfConfig.QuicConfig.MaxIncomingStreams = int64(i)
					return nil
				},
			},
			&cli.DurationFlag{
				Name:  "pile-interval",
				Usage: "the interval to trigger a pile event;\npile up received packets before processing;\n",
				Action: func(ctx *cli.Context, d time.Duration) error {
					if !ctx.IsSet("pile-duration") {
						return fmt.Errorf("pile-interval requires pile-duration")
					}
					config.PileInterval = d
					return nil
				},
			},
			&cli.DurationFlag{
				Name:  "pile-duration",
				Usage: "the duration of a pile event; Pile up received packets before processing.",
				Action: func(ctx *cli.Context, d time.Duration) error {
					if !ctx.IsSet("pile-interval") {
						return fmt.Errorf("pile-duration requires pile-interval")
					}
					config.PileDuration = d
					return nil
				},
			},
		},
		Action: func(c *cli.Context) error {
			if config.PerfConfig.TlsConfig.Certificates == nil {
				fmt.Printf("generate self signed TLS certificate\n")
				config.PerfConfig.TlsConfig.Certificates = []tls.Certificate{common.GenerateCert()}
			}

			win := common.Max(config.PerfConfig.QuicConfig.InitialStreamReceiveWindow, config.PerfConfig.QuicConfig.MaxStreamReceiveWindow)
			config.PerfConfig.QuicConfig.MaxStreamReceiveWindow = win
			config.PerfConfig.QuicConfig.MaxConnectionReceiveWindow = win

			qlogLabel := c.String("log-label")
			config.PerfConfig.QuicConfig.Tracer = qlog2.DefaultConnectionTracer
			config.PerfConfig.QlogLabel = fmt.Sprintf("qperf_%s", qlogLabel)

			addr := common.AppendPortIfNotSpecified(c.String("addr"), c.Int("port"))
			server, err := server.Listen(
				addr,
				config,
			)
			if err != nil {
				return err
			}
			<-server.Context().Done()
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
		CustomAppHelpTemplate: cli.AppHelpTemplate +
			"   $QLOGDIR\toutput directory for qlog files\n",
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
