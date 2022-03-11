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
	"reflect"
	"time"
)

type client struct {
	state                     *common.State
	printRaw                  bool
	reportInterval            time.Duration
	logger                    common.Logger
	lastReportTime            time.Time
	lastReportReceivedBytes   uint64
	lastReportReceivedPackets uint64
}

// Run client.
// if proxyAddr is nil, no proxy is used.
func Run(addr net.UDPAddr, timeToFirstByteOnly bool, printRaw bool, createQLog bool, migrateAfter time.Duration, proxyAddr *net.UDPAddr, probeTime time.Duration, reportInterval time.Duration, tlsServerCertFile string, tlsProxyCertFile string, initialCongestionWindow uint32, initialReceiveWindow uint64, maxReceiveWindow uint64, use0RTT bool, useProxy0RTT, useXse bool, logPrefix string, qlogPrefix string) {
	c := client{
		state:          common.NewState(),
		printRaw:       printRaw,
		reportInterval: reportInterval,
	}

	c.logger = common.DefaultLogger.WithPrefix(logPrefix)

	tracers := make([]logging.Tracer, 0)

	tracers = append(tracers, &common.StateTracer{})

	if createQLog {
		tracers = append(tracers, common.NewQlogTracer(qlogPrefix, c.logger))
	}

	tracers = append(tracers, common.NewMigrationTracer(func(addr net.Addr) {
		c.logger.Infof("migrated to %s", addr)
	}))

	if initialReceiveWindow > maxReceiveWindow {
		maxReceiveWindow = initialReceiveWindow
	}

	var proxyConf *quic.ProxyConfig

	if proxyAddr != nil {
		proxyConf = &quic.ProxyConfig{
			Addr: proxyAddr.String(),
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
		err := common.PingToGatherSessionTicketAndToken(proxyConf.Addr, proxyConf.TlsConf, proxyConf.Config)
		if err != nil {
			panic(fmt.Errorf("failed to prepare 0-RTT to proxy: %w", err))
		}
		c.logger.Infof("stored proxy session ticket and token")
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
		ProxyConf:                                        proxyConf,
		InitialCongestionWindow:                          initialCongestionWindow,
		InitialStreamReceiveWindow:                       initialReceiveWindow,
		MaxStreamReceiveWindow:                           maxReceiveWindow,
		InitialConnectionReceiveWindow:                   uint64(float64(initialReceiveWindow) * quic.ConnectionFlowControlMultiplier),
		MaxConnectionReceiveWindow:                       uint64(float64(maxReceiveWindow) * quic.ConnectionFlowControlMultiplier),
		TokenStore:                                       tokenStore,
		ExtraStreamEncryption:                            useXse,
	}

	if use0RTT {
		err := common.PingToGatherSessionTicketAndToken(addr.String(), tlsConf, &quic.Config{TokenStore: tokenStore})
		if err != nil {
			panic(fmt.Errorf("failed to prepare 0-RTT: %w", err))
		}
		c.logger.Infof("stored session ticket and token")
	}

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

	c.state = common.GetSessionTracerByType(session, reflect.TypeOf(&common.StateConnectionTracer{})).(*common.StateConnectionTracer).State

	go c.reportEstablishmentTime()

	if session.ExtraStreamEncrypted() {
		c.logger.Infof("use XSE-QUIC")
	}

	// migrate
	if migrateAfter.Nanoseconds() != 0 {
		go func() {
			time.Sleep(migrateAfter)
			addr, err := session.MigrateUDPSocket()
			if err != nil {
				panic(fmt.Errorf("failed to migrate UDP socket: %w", err))
			}
			c.logger.Infof("migrated to %s", addr.String())
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

	go c.receive(stream)

	c.reportFirstByte()

	if !timeToFirstByteOnly {
		for {
			if time.Now().Sub(c.state.FirstByteTime()) > probeTime {
				break
			}
			time.Sleep(reportInterval)
			c.report()
		}
	}

	err = session.CloseWithError(common.RuntimeReachedErrorCode, "runtime_reached")
	if err != nil {
		panic(fmt.Errorf("failed to close connection: %w", err))
	}

	c.reportTotal(c.state)
}

func (c *client) reportEstablishmentTime() {
	establishmentTime := c.state.EstablishmentTime().Sub(c.state.StartTime())
	if c.printRaw {
		c.logger.Infof("connection establishment time: %f s",
			establishmentTime.Seconds())
	} else {
		c.logger.Infof("connection establishment time: %s",
			humanize.SIWithDigits(establishmentTime.Seconds(), 2, "s"))
	}
}

func (c *client) reportFirstByte() {
	if c.printRaw {
		c.logger.Infof("time to first byte: %f s",
			c.state.FirstByteTime().Sub(c.state.StartTime()).Seconds())
	} else {
		c.logger.Infof("time to first byte: %s",
			humanize.SIWithDigits(c.state.FirstByteTime().Sub(c.state.StartTime()).Seconds(), 2, "s"))
	}
}

func (c *client) report() {
	now := time.Now()
	totalReceivedBytes := c.state.ReceivedBytes()
	totalReceivedPackets := c.state.ReceivedPackets()
	delta := now.Sub(common.MaxTime([]time.Time{c.lastReportTime, c.state.FirstByteTime()}))
	receivedBytes := totalReceivedBytes - c.lastReportReceivedBytes
	receivedPackets := totalReceivedPackets - c.lastReportReceivedPackets
	if c.printRaw {
		c.logger.Infof("second %f: %f bit/s, bytes received: %d B, packets received: %d",
			time.Now().Sub(c.state.FirstByteTime()).Seconds(),
			float64(receivedBytes)*8/delta.Seconds(),
			receivedBytes,
			receivedPackets)
	} else if c.reportInterval == time.Second {
		c.logger.Infof("second %.0f: %s, bytes received: %s, packets received: %d",
			time.Now().Sub(c.state.FirstByteTime()).Seconds(),
			humanize.SIWithDigits(float64(receivedBytes)*8/delta.Seconds(), 2, "bit/s"),
			humanize.SI(float64(receivedBytes), "B"),
			receivedPackets)
	} else {
		c.logger.Infof("second %.1f: %s, bytes received: %s, packets received: %d",
			time.Now().Sub(c.state.FirstByteTime()).Seconds(),
			humanize.SIWithDigits(float64(receivedBytes)*8/delta.Seconds(), 2, "bit/s"),
			humanize.SI(float64(receivedBytes), "B"),
			receivedPackets)
	}
	c.lastReportTime = now
	c.lastReportReceivedBytes = totalReceivedBytes
	c.lastReportReceivedPackets = totalReceivedPackets
}

func (c *client) reportTotal(state *common.State) {
	receivedBytes, receivedPackets := state.Total()
	if c.printRaw {
		c.logger.Infof("total: bytes received: %d B, packets received: %d",
			receivedBytes,
			receivedPackets)
	} else {
		c.logger.Infof("total: bytes received: %s, packets received: %d",
			humanize.SI(float64(receivedBytes), "B"),
			receivedPackets)
	}
}

func (c *client) receive(reader io.Reader) {
	buf := make([]byte, 65536)
	for {
		_, err := reader.Read(buf)
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
