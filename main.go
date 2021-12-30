package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"net"
	"os"
	"qperf-go/client"
	"qperf-go/common"
	"qperf-go/server"
	"time"
)

func main() {
	app := &cli.App{
		Name:  "qperf",
		Usage: "TODO",
		Commands: []*cli.Command{
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
						Name:  "t",
						Usage: "run for this many seconds",
						Value: 10,
					},
					&cli.StringFlag{
						Name:  "tls-cert",
						Usage: "certificate file to trust the server",
						Value: "server.crt",
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
				},
				Action: func(c *cli.Context) error {
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
						time.Duration(c.Uint("t"))*time.Second,
						c.String("tls-cert"),
						initialReceiveWindow,
						maxReceiveWindow,
						c.Bool("0rtt"),
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
						c.String("tls-cert"),
						c.String("tls-key"),
						initialReceiveWindow,
						maxReceiveWindow,
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
