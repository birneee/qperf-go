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

type Client interface {
	Context() context.Context
}

type client struct {
	conn           quic.Connection
	state          *common.State
	logger         common.Logger
	config         *Config
	qlog           qlog2.QlogWriter
	closeOnce      sync.Once
	qperfCtx       context.Context
	cancelQperfCtx context.CancelFunc
}

func (c *client) Context() context.Context {
	return c.qperfCtx
}

// Run client
func Dial(conf *Config) Client {
	c := &client{
		state:  common.NewState(),
		config: conf,
	}
	c.qperfCtx, c.cancelQperfCtx = context.WithCancel(context.Background())

	c.logger = common.DefaultLogger.WithPrefix(c.config.LogPrefix)

	var tracers []func(ctx context.Context, perspective logging.Perspective, connectionID logging.ConnectionID) logging.ConnectionTracer

	tracers = append(tracers, common.NewStateTracer(c.state))

	if c.config.QlogPathTemplate == "" {
		c.qlog = qlog2.NewStdoutQlogWriter(c.config.QlogConfig)
		tracers = append(tracers, qlog_quic.NewTracer(c.qlog))
	} else {
		tracer := qlog_quic.NewFileQlogTracer(c.config.QlogPathTemplate, c.config.QlogConfig)
		c.qlog = tracer(c.qperfCtx, logging.PerspectiveClient, logging.ConnectionID{}).(qlog_quic.QlogWriterConnectionTracer).QlogWriter()
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
	c.config.QuicConfig.Tracer = common.NewMultiplexedTracer(tracers...)

	if c.config.Use0RTT {
		err := common.PingToGatherSessionTicketAndToken(c.qperfCtx, c.config.RemoteAddress, c.config.TlsConfig, c.config.QuicConfig)
		if err != nil {
			panic(fmt.Errorf("failed to prepare 0-RTT: %w", err))
		}
		c.qlog.RecordEvent(qlog_app.AppInfoEvent{Message: "stored session ticket and token"})
	}

	c.state.SetStartTime()

	go func() {
		for {
			select {
			case <-c.qperfCtx.Done():
				break
			default:
				err := c.runConn()
				if err != nil {
					panic(err)
				}
			}
		}
	}()

	go func() {
		err := c.Run()
		if err != nil {
			panic(err)
		}
	}()

	return c
}

func (c *client) runConn() error {
	c.state.ResetForReconnect()
	quicCtx, cancelQuicCtx := context.WithCancel(c.qperfCtx)
	defer cancelQuicCtx()
	if c.config.Use0RTT {
		var err error
		c.conn, err = quic.DialAddrEarly(quicCtx, c.config.RemoteAddress, c.config.TlsConfig, c.config.QuicConfig)
		if err != nil {
			panic(fmt.Errorf("failed to establish connection: %w", err))
		}
	} else {
		var err error
		c.conn, err = quic.DialAddr(quicCtx, c.config.RemoteAddress, c.config.TlsConfig, c.config.QuicConfig)
		if err != nil {
			panic(fmt.Errorf("failed to establish connection: %w", err))
		}
	}

	go func() {
		c.state.AwaitHandshakeCompleted()
		c.qlog.RecordEventAtTime(c.state.HandshakeCompletedTime(), common.HandshakeCompletedEvent{})
	}()

	go func() {
		c.state.AwaitHandshakeConfirmed()
		c.qlog.RecordEventAtTime(c.state.HandshakeConfirmedTime(), common.HandshakeConfirmedEvent{})
	}()

	stream, err := c.conn.OpenStream()
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
			c.handleError(err, cancelQuicCtx)
		}
		go func() {
			err := c.runSend(stream)
			if err != nil {
				c.handleError(err, cancelQuicCtx)
			}
		}()
	}
	if c.config.SendDatagram {
		go func() {
			err := c.runDatagramSend()
			if err != nil {
				c.handleError(err, cancelQuicCtx)
			}
		}()
	}
	go func() {
		c.state.AwaitFirstByteReceived()
		c.qlog.RecordEventAtTime(c.state.FirstByteReceivedTime(), common.FirstAppDataReceivedEvent{})
	}()
	go func() {
		c.state.AwaitFirstByteSent()
		c.qlog.RecordEventAtTime(c.state.FirstByteSentTime(), common.FirstAppDataSentEvent{})
	}()

	if c.config.TimeToFirstByteOnly {
		c.state.AwaitFirstByteReceived()
	} else {
		go func() {
			stream, err := c.conn.AcceptUniStream(context.Background())
			if err != nil {
				c.handleError(err, cancelQuicCtx)
				return
			}
			err = c.runRawReceive(stream)
			if err != nil {
				c.handleError(err, cancelQuicCtx)
				return
			}
		}()
	}
	<-quicCtx.Done()
	return nil
}

func (c *client) Run() error {

	// close gracefully on interrupt (CTRL+C)
	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt, syscall.SIGTERM, os.Kill)
	go func() {
		<-intChan
		c.Close()
	}()

	if c.config.TimeToFirstByteOnly {
		c.state.AwaitFirstByteReceived()
	} else {

		endTime := c.state.StartTime().Add(c.config.ProbeTime)
		endTimeChan := time.After(endTime.Sub(time.Now()))
	loop:
		for {
			select {
			case <-endTimeChan:
				break loop
			case <-time.After(c.config.ReportInterval):
				c.report(c.state, false)
			case <-c.qperfCtx.Done():
				break loop
			}
		}
	}

	c.Close()
	return nil
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
		c.state.AddReceivedStreamBytes(uint64(read))
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
		c.state.AddSentStreamBytes(uint64(writen))
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

func (c *client) reconnect() {
	var connection quic.Connection
	if c.config.Use0RTT {
		var err error
		connection, err = quic.DialAddrEarly(c.qperfCtx, c.config.RemoteAddress, c.config.TlsConfig, c.config.QuicConfig)
		if err != nil {
			panic(fmt.Errorf("failed to establish connection: %w", err))
		}
	} else {
		var err error
		connection, err = quic.DialAddr(c.qperfCtx, c.config.RemoteAddress, c.config.TlsConfig, c.config.QuicConfig)
		if err != nil {
			panic(fmt.Errorf("failed to establish connection: %w", err))
		}
	}

	c.conn = connection
}

func (c *client) handleError(err error, cancelQuicCtx context.CancelFunc) {
	if c.config.ReconnectOnTimeoutOrReset {
		if _, ok := err.(*quic.IdleTimeoutError); ok {
			cancelQuicCtx()
			return
		}
		if _, ok := err.(*quic.StatelessResetError); ok {
			cancelQuicCtx()
			return
		}
	}
	c.CloseWithError(err)
}

func (c *client) CloseWithError(err error) {
	c.closeOnce.Do(func() {
		if err != nil {
			if _, ok := err.(*quic.IdleTimeoutError); ok {
				// close regularly
			} else if _, ok := err.(*quic.ApplicationError); ok {
				// close regularly
			} else {
				c.logger.Errorf("close with error: %s", err)
			}
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
		c.qlog.Close()
		// flush qlog
		c.cancelQperfCtx()
	})
}

func (c *client) Close() {
	c.CloseWithError(nil)
}
