package client

import (
	"crypto/tls"
	"fmt"
	"github.com/birneee/hquic-proxy-go/proxy"
	"github.com/dustin/go-humanize"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/logging"
	"io"
	"net"
	"os"
	"os/signal"
	"qperf-go/common"
	"time"
)

type client struct {
	state    common.State
	printRaw bool
}

// Run client.
// if proxyAddr is nil, no proxy is used.
func Run(addr net.UDPAddr, timeToFirstByteOnly bool, printRaw bool, createQLog bool, migrateAfter time.Duration, proxyAddr *net.UDPAddr, probeTime time.Duration, tlsServerCertFile string, tlsProxyCertFile string, initialCongestionWindow uint32, initialReceiveWindow uint64, maxReceiveWindow uint64, use0RTT bool, useProxy0RTT, useXse bool) {
	c := client{
		state:    common.State{},
		printRaw: printRaw,
	}

	logger := common.DefaultLogger.Clone()
	if len(os.Getenv(common.LogEnv)) == 0 {
		logger.SetLogLevel(common.LogLevelInfo) // log level info is the default
	}

	tracers := make([]logging.Tracer, 0)

	tracers = append(tracers, common.StateTracer{
		State: &c.state,
	})

	if createQLog {
		tracers = append(tracers, common.NewQlogTrager("client", logger))
	}

	tracers = append(tracers, common.NewMigrationTracer(func(addr net.Addr) {
		fmt.Printf("migrated to %s\n", addr)
	}))

	if initialReceiveWindow > maxReceiveWindow {
		maxReceiveWindow = initialReceiveWindow
	}

	var proxyConf *quic.ProxyConfig

	if proxyAddr != nil {
		proxyConf = &quic.ProxyConfig{
			Addr: proxyAddr,
			TlsConf: &tls.Config{
				RootCAs:            common.NewCertPoolWithCert(tlsProxyCertFile),
				NextProtos:         []string{proxy.HQUICProxyALPN},
				ClientSessionCache: tls.NewLRUClientSessionCache(1),
			},
			Config: &quic.Config{
				LoggerPrefix:          "proxy control",
				TokenStore:            quic.NewLRUTokenStore(1, 1),
				EnableActiveMigration: true,
			},
		}
	}

	if useProxy0RTT {
		err := common.PingToGatherSessionTicketAndToken(proxyConf.Addr.String(), proxyConf.TlsConf, proxyConf.Config)
		if err != nil {
			panic(fmt.Errorf("failed to prepare 0-RTT to proxy: %w", err))
		}
		fmt.Printf("stored proxy session ticket and token\n")
	}

	var clientSessionCache tls.ClientSessionCache
	if use0RTT {
		clientSessionCache = tls.NewLRUClientSessionCache(1)
	}

	var tokenStore quic.TokenStore
	if use0RTT {
		tokenStore = quic.NewLRUTokenStore(1, 1)
	}

	tlsConf := &tls.Config{
		RootCAs:            common.NewCertPoolWithCert(tlsServerCertFile),
		NextProtos:         []string{common.QperfALPN},
		ClientSessionCache: clientSessionCache,
	}

	conf := quic.Config{
		Tracer: logging.NewMultiplexedTracer(tracers...),
		IgnoreReceived1RTTPacketsUntilFirstPathMigration: proxyAddr != nil, // TODO maybe not necessary for client
		EnableActiveMigration:                            true,
		Proxy:                                            proxyConf,
		InitialCongestionWindow:                          initialCongestionWindow,
		InitialStreamReceiveWindow:                       initialReceiveWindow,
		MaxStreamReceiveWindow:                           maxReceiveWindow,
		InitialConnectionReceiveWindow:                   uint64(float64(initialReceiveWindow) * quic.ConnectionFlowControlMultiplier),
		MaxConnectionReceiveWindow:                       uint64(float64(maxReceiveWindow) * quic.ConnectionFlowControlMultiplier),
		TokenStore:                                       tokenStore,
		ExtraStreamEncryption:                            useXse,
	}

	if use0RTT {
		err := common.PingToGatherSessionTicketAndToken(addr.String(), tlsConf, &conf)
		if err != nil {
			panic(fmt.Errorf("failed to prepare 0-RTT: %w", err))
		}
		fmt.Printf("stored session ticket and token\n")
	}

	c.state.SetStartTime()

	var session quic.Session
	if use0RTT {
		var err error
		session, err = quic.DialAddrEarly(addr.String(), tlsConf, &conf)
		if err != nil {
			panic(fmt.Errorf("failed to establish connection: %w", err))
		}
	} else {
		var err error
		session, err = quic.DialAddr(addr.String(), tlsConf, &conf)
		if err != nil {
			panic(fmt.Errorf("failed to establish connection: %w", err))
		}
	}

	c.state.SetEstablishmentTime()
	c.reportEstablishmentTime(&c.state)

	if session.ExtraStreamEncrypted() {
		println("use XSE-QUIC")
	}

	// migrate
	if migrateAfter.Nanoseconds() != 0 {
		go func() {
			time.Sleep(migrateAfter)
			addr, err := session.MigrateUDPSocket()
			if err != nil {
				panic(fmt.Errorf("failed to migrate UDP socket: %w", err))
			}
			fmt.Printf("migrated to %s\n", addr.String())
		}()
	}

	// close gracefully on interrupt (CTRL+C)
	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt)
	go func() {
		<-intChan
		_ = session.CloseWithError(quic.ApplicationErrorCode(quic.NoError), "client_closed")
		os.Exit(0)
	}()

	stream, err := session.OpenStream()
	if err != nil {
		panic(fmt.Errorf("failed to open stream: %w", err))
	}

	// send some date to open stream
	_, err = stream.Write([]byte(common.QPerfStartSendingRequest))
	if err != nil {
		panic(fmt.Errorf("failed to write to stream: %w", err))
	}

	err = c.receiveFirstByte(stream)
	if err != nil {
		panic(fmt.Errorf("failed to receive first byte: %w", err))
	}

	c.reportFirstByte(&c.state)

	if !timeToFirstByteOnly {
		go c.receive(stream)

		for {
			if time.Now().Sub(c.state.GetFirstByteTime()) > probeTime {
				break
			}
			time.Sleep(1 * time.Second)
			c.report(&c.state)
		}
	}

	err = session.CloseWithError(common.RuntimeReachedErrorCode, "runtime_reached")
	if err != nil {
		panic(fmt.Errorf("failed to close connection: %w", err))
	}

	c.reportTotal(&c.state)
}

func (c *client) reportEstablishmentTime(state *common.State) {
	establishmentTime := state.EstablishmentTime().Sub(state.StartTime())
	if c.printRaw {
		fmt.Printf("connection establishment time: %f s\n",
			establishmentTime.Seconds())
	} else {
		fmt.Printf("connection establishment time: %s\n",
			humanize.SIWithDigits(establishmentTime.Seconds(), 2, "s"))
	}
}

func (c *client) reportFirstByte(state *common.State) {
	if c.printRaw {
		fmt.Printf("time to first byte: %f s\n",
			state.GetFirstByteTime().Sub(state.StartTime()).Seconds())
	} else {
		fmt.Printf("time to first byte: %s\n",
			humanize.SIWithDigits(state.GetFirstByteTime().Sub(state.StartTime()).Seconds(), 2, "s"))
	}
}

func (c *client) report(state *common.State) {
	receivedBytes, receivedPackets, delta := state.GetAndResetReport()
	if c.printRaw {
		fmt.Printf("second %f: %f bit/s, bytes received: %d B, packets received: %d\n",
			time.Now().Sub(state.GetFirstByteTime()).Seconds(),
			float64(receivedBytes)*8/delta.Seconds(),
			receivedBytes,
			receivedPackets)
	} else {
		fmt.Printf("second %.2f: %s, bytes received: %s, packets received: %d\n",
			time.Now().Sub(state.GetFirstByteTime()).Seconds(),
			humanize.SIWithDigits(float64(receivedBytes)*8/delta.Seconds(), 2, "bit/s"),
			humanize.SI(float64(receivedBytes), "B"),
			receivedPackets)
	}
}

func (c *client) reportTotal(state *common.State) {
	receivedBytes, receivedPackets := state.Total()
	if c.printRaw {
		fmt.Printf("total: bytes received: %d B, packets received: %d\n",
			receivedBytes,
			receivedPackets)
	} else {
		fmt.Printf("total: bytes received: %s, packets received: %d\n",
			humanize.SI(float64(receivedBytes), "B"),
			receivedPackets)
	}
}

func (c *client) receiveFirstByte(stream quic.ReceiveStream) error {
	buf := make([]byte, 1)
	for {
		received, err := stream.Read(buf)
		if err != nil {
			return err
		}
		if received != 0 {
			c.state.AddReceivedBytes(uint64(received))
			return nil
		}
	}
}

func (c *client) receive(reader io.Reader) {
	buf := make([]byte, 65536)
	for {
		received, err := reader.Read(buf)
		c.state.AddReceivedBytes(uint64(received))
		if err != nil {
			switch err := err.(type) {
			case *quic.ApplicationError:
				if err.ErrorCode == common.RuntimeReachedErrorCode {
					return
				}
			default:
				panic(err)
			}
		}
	}
}
