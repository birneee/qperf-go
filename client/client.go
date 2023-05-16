package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"io"
	"os"
	"os/signal"
	"qperf-go/common"
	qlog2 "qperf-go/common/qlog"
	"qperf-go/common/qlog_app"
	"qperf-go/common/qlog_quic"
	"qperf-go/control_frames"
	"qperf-go/errors"
	"sync"
	"syscall"
	"time"
)

type client struct {
	conn      quic.Connection
	state     *common.State
	logger    common.Logger
	config    *Config
	qlog      qlog2.QlogWriter
	closeOnce sync.Once
	ctx       context.Context
	cancelCtx context.CancelFunc
}

// Run client
func Run(conf *Config) {
	c := client{
		state:  common.NewState(),
		config: conf,
	}
	c.ctx, c.cancelCtx = context.WithCancel(context.Background())

	c.logger = common.DefaultLogger.WithPrefix(c.config.LogPrefix)

	var tracers []func(ctx context.Context, perspective logging.Perspective, connectionID logging.ConnectionID) logging.ConnectionTracer

	tracers = append(tracers, common.NewStateTracer(c.state))

	if c.config.QlogPathTemplate == "" {
		c.qlog = qlog2.NewStdoutQlogWriter(c.config.QlogConfig)
		tracers = append(tracers, qlog_quic.NewTracer(c.qlog))
	} else {
		tracer := qlog_quic.NewFileQlogTracer(c.config.QlogPathTemplate, c.config.QlogConfig)
		c.qlog = tracer(c.ctx, logging.PerspectiveClient, logging.ConnectionID{}).(qlog_quic.QlogWriterConnectionTracer).QlogWriter()
		tracers = append(tracers, tracer)
	}

	if c.config.TlsConfig.ClientSessionCache != nil {
		panic("unexpected value")
	}
	if c.config.Use0RTT {
		c.config.TlsConfig.ClientSessionCache = tls.NewLRUClientSessionCache(1)
	}

	if c.config.QuicConfig.TokenStore != nil {
		panic("unexpected value")
	}
	if c.config.Use0RTT {
		c.config.QuicConfig.TokenStore = quic.NewLRUTokenStore(1, 1)
	}

	if c.config.QuicConfig.Tracer != nil {
		panic("unexptected value")
	}
	c.config.QuicConfig.Tracer = func(ctx context.Context, perspective logging.Perspective, id quic.ConnectionID) logging.ConnectionTracer {
		var connectionTracers []logging.ConnectionTracer
		for _, tracer := range tracers {
			connectionTracers = append(connectionTracers, tracer(ctx, perspective, id))
		}
		return logging.NewMultiplexedConnectionTracer(connectionTracers...)
	}

	if c.config.Use0RTT {
		err := common.PingToGatherSessionTicketAndToken(c.ctx, c.config.RemoteAddress, c.config.TlsConfig, c.config.QuicConfig)
		if err != nil {
			panic(fmt.Errorf("failed to prepare 0-RTT: %w", err))
		}
		c.qlog.RecordEvent(qlog_app.AppInfoEvent{Message: "stored session ticket and token"})
	}

	c.state.SetStartTime()

	var connection quic.Connection
	if c.config.Use0RTT {
		var err error
		connection, err = quic.DialAddrEarly(c.ctx, c.config.RemoteAddress, c.config.TlsConfig, c.config.QuicConfig)
		if err != nil {
			panic(fmt.Errorf("failed to establish connection: %w", err))
		}
	} else {
		var err error
		connection, err = quic.DialAddr(c.ctx, c.config.RemoteAddress, c.config.TlsConfig, c.config.QuicConfig)
		if err != nil {
			panic(fmt.Errorf("failed to establish connection: %w", err))
		}
	}

	c.conn = connection

	//TODO extract somehow from connection tracer
	c.state.SetEstablishmentTime()
	if c.qlog != nil {
		c.qlog.RecordEventAtTime(c.state.EstablishmentTime(), common.HandshakeCompletedEvent{})
	}

	go func() {
		c.state.AwaitHandshakeConfirmed()
		c.qlog.RecordEventAtTime(c.state.HandshakeConfirmedTime(), common.HandshakeConfirmedEvent{})
	}()

	// close gracefully on interrupt (CTRL+C)
	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt)
	go func() {
		<-intChan
		_ = connection.CloseWithError(quic.ApplicationErrorCode(quic.NoError), "client_closed")
		os.Exit(0)
	}()

	stream, err := connection.OpenStream()
	if err != nil {
		panic(fmt.Errorf("failed to open stream: %w", err))
	}

	frameStream := control_frames.NewControlFrameStream(stream)

	if c.config.ReceiveStream {
		err = frameStream.WriteFrame(&control_frames.StartSendingFrame{StreamID: stream.StreamID()})
		if err != nil {
			panic(fmt.Errorf("failed to write frame: %w", err))
		}
	}

	if c.config.ReceiveDatagram {
		err = frameStream.WriteFrame(&control_frames.StartSendingDatagramsFrame{})
		if err != nil {
			panic(fmt.Errorf("failed to write frame: %w", err))
		}
	}

	if c.config.SendStream {
		stream, err := c.conn.OpenUniStream()
		if err != nil {
			c.CloseWithError(err)
		}
		go func() {
			err := c.runSend(stream)
			if err != nil {
				c.CloseWithError(err)
			}
		}()
	}
	if c.config.SendDatagram {
		go func() {
			err := c.runDatagramSend()
			if err != nil {
				c.CloseWithError(err)
			}
		}()
	}
	go func() {
		c.state.AwaitFirstByteReceived()
		c.qlog.RecordEventAtTime(c.state.FirstByteTime(), common.FirstAppDataReceivedEvent{})
	}()

	if c.config.TimeToFirstByteOnly {
		c.state.AwaitFirstByteReceived()
	} else {
		go func() {
			stream, err := c.conn.AcceptUniStream(context.Background())
			if err != nil {
				c.CloseWithError(err)
				return
			}
			err = c.runRawReceive(stream)
			//err := c.runFrameReceive(frameStream)
			if err != nil {
				c.CloseWithError(err)
				return
			}
		}()

		intChan := make(chan os.Signal, 1)
		signal.Notify(intChan, os.Interrupt, syscall.SIGTERM, os.Kill)
		go func() {
			<-intChan
			c.Close()
		}()

		for {
			if time.Now().Sub(c.state.StartTime()) > c.config.ProbeTime {
				break
			}
			select {
			case <-time.After(c.config.ReportInterval):
				c.report(c.state, false)
			case <-c.ctx.Done():
				break
			}
		}
	}

	c.Close()
}

func (c *client) report(state *common.State, total bool) {
	var report common.Report
	if total {
		report = state.TotalReport()
	} else {
		report = state.GetAndResetReport()
	}
	now := time.Now()
	event := &common.ReportEvent{
		Period: report.TimeAggregated,
		//PacketsReceived:   &report.ReceivedPackets,
		//MinRTT: &report.MinRTT,
	}
	if c.config.ReportMaxRTT {
		event.MaxRTT = &report.MaxRTT
	}
	if c.config.ReportLostPackets {
		event.PacketsLost = &report.PacketsLost
	}
	if c.config.ReceiveStream {
		mbps := float32(report.ReceivedBytes) * 8 / float32(report.TimeAggregated.Seconds()) / float32(1e6)
		event.StreamMegaBitsPerSecondReceived = &mbps
		event.StreamBytesReceived = &report.ReceivedBytes
	}
	if c.config.ReceiveDatagram {
		mbps := float32(report.ReceivedDatagramBytes) * 8 / float32(report.TimeAggregated.Seconds()) / float32(1e6)
		event.DatagramMegaBitsPerSecondReceived = &mbps
		event.DatagramBytesReceived = &report.ReceivedDatagramBytes
	}
	if c.config.SendStream {
		mbps := float32(report.SentBytes) * 8 / float32(report.TimeAggregated.Seconds()) / float32(1e6)
		event.StreamMegaBitsPerSecondSent = &mbps
		event.StreamBytesSent = &report.SentBytes
	}
	if c.config.SendDatagram {
		mbps := float32(report.SentDatagramBytes) * 8 / float32(report.TimeAggregated.Seconds()) / float32(1e6)
		event.DatagramMegaBitsPerSecondSent = &mbps
		event.DatagramBytesSent = &report.SentDatagramBytes
	}
	if total {
		c.qlog.RecordEventAtTime(now, common.TotalEvent{ReportEvent: *event})
	} else {
		c.qlog.RecordEventAtTime(now, event)
	}
}

// do not interpret qperf control_frames
func (c *client) runRawReceive(stream quic.ReceiveStream) error {
	for {
		read, err := io.CopyN(io.Discard, stream, int64(control_frames.MaxFrameLength))
		c.state.AddReceivedBytes(uint64(read))
		if err != nil {
			return err
		}
	}
}

func (c *client) runSend(stream quic.SendStream) error {
	var buf [65536]byte
	for {
		writen, err := stream.Write(buf[:])
		if err != nil {
			return err
		}
		c.state.AddSentBytes(uint64(writen))
	}
}

func (c *client) runDatagramSend() error {
	var buf = make([]byte, 1197)
	//TODO calculate size from max_datagram_frame_size, max_udp_payload_size and path MTU; https://github.com/quic-go/quic-go/issues/3599
	for {
		err := c.conn.SendMessage(buf[:])
		if err != nil {
			return err
		}
	}
}

func (c *client) CloseWithError(err error) {
	c.closeOnce.Do(func() {
		if err != nil {
			c.logger.Errorf("close with error: %s", err)
			err := c.conn.CloseWithError(errors.InternalErrorCode, "internal error")
			if err != nil {
				panic(fmt.Errorf("failed to close connection: %s", err))
			}
		} else {
			err := c.conn.CloseWithError(errors.NoError, "no error")
			if err != nil {
				panic(fmt.Errorf("failed to close connection: %w", err))
			}
		}
		c.report(c.state, true)
		c.cancelCtx()
		c.qlog.Close()
		// flush qlog
	})
}

func (c *client) Close() {
	c.CloseWithError(nil)
}
