package main

import (
	"github.com/urfave/cli/v2"
	"net"
	"os"
	"qperf-go/client"
	"qperf-go/server"
)

const addr = "localhost:4242"

func main() {
	app := &cli.App{
		Name: "qperf",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "s",
				Usage: "run in server mode",
			},
			&cli.StringFlag{
				Name:  "c",
				Usage: "run in client mode and connect to target server",
			},
			&cli.BoolFlag{
				Name:  "e",
				Usage: "measure time for connection establishment and first byte only",
			},
			&cli.BoolFlag{
				Name:  "print-raw",
				Usage: "output raw statistics, don't calculate metric prefixes",
			},
			&cli.UintFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "port to connect to",
				Value:   18080,
			},
			&cli.StringFlag{
				Name:  "listen-addr",
				Usage: "address to listen on in server/proxy mode",
				Value: "0.0.0.0",
			},
			&cli.UintFlag{
				Name:  "listen-port",
				Usage: "port to listen on in server/proxy mode",
				Value: 18080,
			},
			&cli.BoolFlag{
				Name:  "qlog",
				Usage: "create qlog file",
			},
		},
		//todo use addr and port values
		Action: func(c *cli.Context) error {
			if c.Bool("s") == true {
				server.Run(net.UDPAddr{
					IP:   net.ParseIP(c.String("listen-addr")),
					Port: c.Int("listen-port"),
				},
					c.Bool("qlog"))
			} else if c.IsSet("c") {
				client.Run(net.UDPAddr{
					IP:   net.ParseIP(c.String("c")),
					Port: c.Int("port"),
				},
					c.Bool("e"),
					c.Bool("print-raw"),
					c.Bool("qlog"))
			} else {
				println("exactly one mode must be stated")
				cli.ShowAppHelpAndExit(c, 1)
			}
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
