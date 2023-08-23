package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"os"
	"os/signal"
	"qperf-go/common"
	qlog2 "qperf-go/common/qlog"
	"qperf-go/common/qlog_app"
	"qperf-go/common/qlog_quic"
	"qperf-go/perf"
	"qperf-go/perf/perf_client"
	"sync"
	"syscall"
	"time"
)

type Client interface {
	Context() context.Context
}

type client struct {
	perfClient perf_client.Client
	state      *common.State
	config     *Config
	qlog       qlog2.Writer
	closeOnce  sync.Once
	// closed when client is stopping and doing some final output and cleanup
	stopping       chan struct{}
	streamLoopDone chan struct{}
	qperfCtx       context.Context
	cancelQperfCtx context.CancelFunc
}

func (c *client) Context() context.Context {
	return c.qperfCtx
}

// Dial starts a new client
func Dial(conf *Config) Client {
	c := &client{
		state:          common.NewState(),
		config:         conf,
		stopping:       make(chan struct{}),
		streamLoopDone: make(chan struct{}),
	}
	c.qperfCtx, c.cancelQperfCtx = context.WithCancel(context.Background())

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
			case <-c.stopping:
				break
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
	if c.config.Use0RTT {
		var err error
		c.perfClient, err = perf_client.DialEarlyAddr(c.config.RemoteAddress, &perf_client.Config{
			QuicConfig: c.config.QuicConfig,
			TlsConfig:  c.config.TlsConfig,
		})
		if err != nil {
			return err
		}
	} else {
		var err error
		c.perfClient, err = perf_client.DialAddr(c.config.RemoteAddress, &perf_client.Config{
			QuicConfig: c.config.QuicConfig,
			TlsConfig:  c.config.TlsConfig,
		})
		if err != nil {
			return err
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

	if c.perfClient.ExtraApplicationDataSecurity() {
		c.qlog.RecordEvent(qlog_app.AppInfoEvent{Message: "use XADS-QUIC"})
	}

	c.runStreamRequestLoop()

	if c.config.ReceiveInfiniteStream {
		_, _, err := c.perfClient.Request(0, perf.MaxResponseLength, 0)
		if err != nil {
			c.handlePerfClose(err)
		}
	}

	if c.config.SendInfiniteStream {
		_, _, err := c.perfClient.Request(perf.MaxRequestLength, 0, 0)
		if err != nil {
			c.handlePerfClose(err)
		}
	}

	if c.config.ReceiveDatagram {
		err := c.perfClient.DatagramRequest(0, perf.MaxDatagramResponseNum, perf.MaxDatagramResponseLength, 0)
		if err != nil {
			c.handlePerfClose(err)
		}

	}

	if c.config.SendDatagram {
		go func() {
			for {
				err := c.perfClient.DatagramRequest(perf.MaxDatagramRequestLength, 0, 0, 0)
				if err != nil {
					c.handlePerfClose(err)
				}
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

	<-c.perfClient.Context().Done()
	err := c.perfClient.Close()
	c.handlePerfClose(err)
	return nil
}

func (c *client) runStreamRequestLoop() {
	if c.config.RequestLength == 0 && c.config.ResponseLength == 0 {
		close(c.streamLoopDone)
		return
	}
	go func() {
	requestLoop:
		for {
			go func() {
				req, resp, err := c.perfClient.Request(c.config.RequestLength, c.config.ResponseLength, c.config.ResponseDelay)
				if err != nil {
					c.handlePerfClose(err)
					return
				}
				select {
				case <-req.Context().Done():
				case <-time.After(c.config.Deadline):
					req.Cancel()
				}
				if resp != nil {
					select {
					case <-resp.Context().Done():
						if resp.Success() {
							c.state.AddReceivedResponses(1)
						}
					case <-time.After(c.config.Deadline):
						resp.Cancel()
						c.state.AddDeadlineExceededResponses(1)
					}
				}
				if c.config.RequestInterval == 0 {
					close(c.streamLoopDone)
				}
			}()
			if c.config.RequestInterval == 0 {
				break requestLoop
			}
			time.Sleep(c.config.RequestInterval)
		}
	}()
}

func (c *client) Run() error {

	// close gracefully on interrupt (CTRL+C)
	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt, syscall.SIGTERM, os.Kill)
	go func() {
		<-intChan
		c.Close()
	}()

	go func() {
		<-c.streamLoopDone
		if !c.config.ReceiveInfiniteStream && !c.config.SendInfiniteStream && !c.config.ReceiveDatagram && !c.config.SendDatagram {
			c.Close()
		}
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
	if c.perfClient != nil {
		state.SetTotalReceiveStreamBytes(c.perfClient.ReceivedBytes())
		state.SetTotalSentStreamBytes(c.perfClient.SentBytes())
	}
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
	if c.config.ResponseLength != 0 || c.config.ReceiveInfiniteStream {
		mbps := float32(report.ReceivedBytes) * 8 / float32(report.TimeAggregated.Seconds()) / float32(1e6)
		event.StreamMegaBitsPerSecondReceived = &mbps
		event.StreamBytesReceived = &report.ReceivedBytes
	}
	if c.config.ReceiveDatagram {
		mbps := float32(report.ReceivedDatagramBytes) * 8 / float32(report.TimeAggregated.Seconds()) / float32(1e6)
		event.DatagramMegaBitsPerSecondReceived = &mbps
		event.DatagramBytesReceived = &report.ReceivedDatagramBytes
	}
	if c.config.RequestLength != 0 || c.config.SendInfiniteStream {
		mbps := float32(report.SentBytes) * 8 / float32(report.TimeAggregated.Seconds()) / float32(1e6)
		event.StreamMegaBitsPerSecondSent = &mbps
		event.StreamBytesSent = &report.SentBytes
	}
	if c.config.SendDatagram {
		mbps := float32(report.SentDatagramBytes) * 8 / float32(report.TimeAggregated.Seconds()) / float32(1e6)
		event.DatagramMegaBitsPerSecondSent = &mbps
		event.DatagramBytesSent = &report.SentDatagramBytes
	}
	if c.config.RequestInterval != 0 {
		event.ResponsesReceived = &report.ReceivedResponses
	}
	if report.DeadlineExceededResponses != 0 {
		event.DeadlineExceededResponses = &report.DeadlineExceededResponses
	}
	if total {
		c.qlog.RecordEventAtTime(now, common.TotalEvent{ReportEvent: *event})
	} else {
		c.qlog.RecordEventAtTime(now, event)
	}
}

func (c *client) handlePerfClose(err error) {
	if c.config.ReconnectOnTimeoutOrReset {
		if _, ok := err.(*quic.IdleTimeoutError); ok {
			return // reconnect
		}
		if _, ok := err.(*quic.StatelessResetError); ok {
			return // reconnect
		}
	}
	c.CloseWithError(err)
}

func (c *client) CloseWithError(err error) {
	c.closeOnce.Do(func() {
		close(c.stopping)
		if err != nil {
			if _, ok := err.(*quic.IdleTimeoutError); ok {
				// close regularly
			} else if _, ok := err.(*quic.ApplicationError); ok {
				// close regularly
			} else {
				panic(fmt.Errorf("close with error: %s", err).Error())
			}
			c.perfClient.Close()
		} else {
			c.perfClient.Close()
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
