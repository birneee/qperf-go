package client

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"github.com/quic-go/quic-go/qlog"
	"net"
	"os"
	"os/signal"
	"qperf-go/common"
	qlog2 "qperf-go/common/qlog"
	"qperf-go/common/qlog_app"
	"qperf-go/perf"
	"qperf-go/perf/perf_client"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Client interface {
	// Context is done when all tasks are finished or Close is called manually
	Context() context.Context
	TotalReport() common.Report
	Close()
}

type client struct {
	totalReceivedStreamBytesByPreviousPerfConns uint64
	totalSentStreamBytesByPreviousPerfConns     uint64
	perfClient                                  perf_client.Client
	state                                       *common.State
	config                                      *Config
	qlog                                        qlog2.Writer
	closeOnce                                   sync.Once
	// closed when client is stopping and doing some final output and cleanup
	stopping       chan struct{}
	streamLoopDone chan struct{}
	qperfCtx       context.Context
	cancelQperfCtx context.CancelFunc
	// number of stream requests and responses that are processed, either successfully or by a deadline
	finishedStreamRequests atomic.Uint64
	// number of stream requests that have been started
	startedStreamRequests atomic.Uint64
	// closed when first perf client is ready to use
	perfClientReady chan struct{}
	// closed when the reconnect loop has stopped
	reconnectLoopDone chan struct{}
	// closed when the report loop has stopped
	reportLoopDone chan struct{}
}

func (c *client) Context() context.Context {
	return c.qperfCtx
}

// Dial starts a new client
func Dial(conf *Config) Client {
	c := &client{
		state:             common.NewState(),
		config:            conf.Populate(),
		stopping:          make(chan struct{}),
		streamLoopDone:    make(chan struct{}),
		perfClientReady:   make(chan struct{}),
		reconnectLoopDone: make(chan struct{}),
		reportLoopDone:    make(chan struct{}),
	}
	c.qperfCtx, c.cancelQperfCtx = context.WithCancel(context.Background())

	if c.qlog == nil {
		var id [4]byte
		rand.Read(id[:])
		c.qlog = qlog2.NewQlogDirWriter(id[:], "qperf_client", c.config.QlogConfig)
	}

	if c.qlog == nil {
		c.qlog = qlog2.NewStdoutQlogWriter(c.config.QlogConfig)
	}

	var tracers []func(ctx context.Context, perspective logging.Perspective, connectionID logging.ConnectionID) *logging.ConnectionTracer

	if c.config.QuicConfig.Tracer != nil {
		tracers = append(tracers, c.config.QuicConfig.Tracer)
	}

	tracers = append(tracers, qlog.DefaultConnectionTracer)

	tracers = append(tracers, common.NewStateTracer(c.state).TracerForConnection)

	tracers = append(tracers, func(_ context.Context, _ logging.Perspective, _ logging.ConnectionID) *logging.ConnectionTracer {
		return &logging.ConnectionTracer{
			StartedConnection: func(_, _ net.Addr, _, destConnID logging.ConnectionID) {
				c.qlog.RecordEvent(common.EventConnectionStarted{DestConnectionID: destConnID})
			},
			ClosedConnection: func(err error) {
				c.qlog.RecordEvent(common.EventConnectionClosed{Err: err})

			},
			Debug: func(name, msg string) {
				c.qlog.RecordEvent(common.EventGeneric{CategoryF: "transport", NameF: name, MsgF: msg})
			},
		}
	})

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

	c.config.QuicConfig.Tracer = common.NewMultiplexedTracer(tracers...)

	if c.config.Use0RTT {
		err := common.PingToGatherSessionTicketAndToken(
			c.qperfCtx,
			c.config.RemoteAddress,
			c.config.TlsConfig.ClientSessionCache,
			c.config.QuicConfig.TokenStore,
			c.config.TlsConfig.NextProtos[0],
			c.config.TlsConfig.RootCAs,
			c.config.TlsConfig.ServerName,
			qlog.DefaultConnectionTracer,
		)
		if err != nil {
			panic(fmt.Errorf("failed to prepare 0-RTT: %w", err))
		}
		c.qlog.RecordEvent(qlog_app.AppInfoEvent{Message: "stored session ticket and token"})
	}

	c.state.SetStartTime()

	go func() {
	reconnectLoop:
		for {
			select {
			case <-c.stopping:
				break reconnectLoop
			default:
				err := c.runConn()
				if err != nil {
					panic(err)
				}
			}
		}
		close(c.reconnectLoopDone)
	}()

	go func() {
		c.runRequestLoop()
		close(c.streamLoopDone)
	}()

	go func() {
		err := c.Run()
		if err != nil {
			c.close(err)
		}
		close(c.reportLoopDone)
	}()

	return c
}

func (c *client) runConn() error {
	if c.perfClient != nil {
		c.state.ResetForReconnect()
		c.qlog.RecordEvent(qlog_app.AppInfoEvent{Message: "reconnect"})
		c.totalSentStreamBytesByPreviousPerfConns += c.perfClient.SentBytes()
		c.totalReceivedStreamBytesByPreviousPerfConns += c.perfClient.ReceivedBytes()
	}
	var err error
	c.perfClient, err = perf_client.DialAddr(
		c.config.RemoteAddress,
		&perf_client.Config{
			QuicConfig: c.config.QuicConfig,
			TlsConfig:  c.config.TlsConfig,
			Qlog:       c.qlog,
		},
		c.config.Use0RTT)
	if err != nil {
		return err
	}

	select {
	case <-c.perfClientReady:
	default:
		close(c.perfClientReady)
	}

	go func() {
		c.state.AwaitHandshakeCompleted()
		c.qlog.RecordEventAtTime(c.state.HandshakeCompletedTime(), common.HandshakeCompletedEvent{})
	}()

	go func() {
		c.state.AwaitHandshakeConfirmed()
		c.qlog.RecordEventAtTime(c.state.HandshakeConfirmedTime(), common.HandshakeConfirmedEvent{})
	}()

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
		panic("implement me")
	}

	if c.config.SendDatagram {
		panic("implement me")
	}
	go func() {
		c.state.AwaitFirstByteReceived()
		c.qlog.RecordEventAtTime(c.state.FirstByteReceivedTime(), common.FirstAppDataReceivedEvent{})
	}()
	go func() {
		c.state.AwaitFirstByteSent()
		c.qlog.RecordEventAtTime(c.state.FirstByteSentTime(), common.FirstAppDataSentEvent{})
	}()

	select {
	case <-c.perfClient.Context().Done():
	case <-c.stopping:
	}
	err = c.perfClient.Close()
	c.handlePerfClose(err)
	return nil
}

func (c *client) runRequestLoop() {
	if c.config.RequestLength == 0 && c.config.ResponseLength == 0 {
		return
	}
	<-c.perfClientReady
	var respWG sync.WaitGroup
requestLoop:
	for {
		respWG.Add(1)
		go func() {
			defer c.finishedStreamRequests.Add(1)
			defer respWG.Done()
			req, resp, err := c.perfClient.Request(c.config.RequestLength, c.config.ResponseLength, c.config.ResponseDelay)
			if err != nil {
				c.handlePerfClose(err)
				return // cancel current request
			}
			select {
			case <-req.Context().Done():
			case <-time.After(c.config.RequestDeadline):
				req.Cancel()
				resp.Cancel()
				// exceeded deadlines will be counted in the next step
			}
			select {
			case <-resp.Context().Done():
				if resp.Success() {
					c.state.AddReceivedResponses(1)
				}
			case <-time.After(c.config.ResponseDeadline):
				req.Cancel()
				resp.Cancel()
				c.state.AddDeadlineExceededResponses(1)
			}
		}()
		c.startedStreamRequests.Add(1)
		if c.startedStreamRequests.Load() >= c.config.NumRequests {
			break requestLoop
		}
		select {
		case <-c.stopping:
			break requestLoop
		default:
		}
		time.Sleep(c.config.RequestInterval)
	}
	respWG.Wait()
}

func (c *client) Run() error {

	// close gracefully on interrupt (CTRL+C)
	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt, syscall.SIGTERM, os.Kill)
	go func() {
		<-intChan
		c.Close()
	}()

	if !c.config.SendInfiniteStream && !c.config.ReceiveInfiniteStream && !c.config.ReceiveDatagram && !c.config.SendDatagram {
		go func() {
			<-c.streamLoopDone
			c.Close()
		}()
	}

	if c.config.TimeToFirstByteOnly {
		c.state.AwaitFirstByteReceived()
	} else {

		endTime := c.state.StartTime().Add(c.config.ProbeTime)
		endTimeChan := time.After(endTime.Sub(time.Now()))
	loop:
		for {
			select {
			case <-time.After(c.config.ReportInterval):
				c.report(c.state, false)
			case <-endTimeChan:
				break loop
			case <-c.stopping:
				break loop
			}
		}
	}

	c.close(nil)
	return nil
}

func (c *client) report(state *common.State, total bool) {
	var report common.Report
	if c.perfClient != nil {
		state.SetTotalReceiveStreamBytes(c.totalReceivedStreamBytesByPreviousPerfConns + c.perfClient.ReceivedBytes())
		state.SetTotalSentStreamBytes(c.totalSentStreamBytesByPreviousPerfConns + c.perfClient.SentBytes())
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
	if c.config.NumRequests > 1 {
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
	c.close(err)
}

func (c *client) close(err error) {
	c.closeOnce.Do(func() {
		close(c.stopping)
		if err != nil {
			if _, ok := err.(*quic.IdleTimeoutError); ok {
				// close regularly
			} else if _, ok := err.(*quic.ApplicationError); ok {
				// close regularly
			} else if _, ok := err.(*quic.StatelessResetError); ok {
				c.perfClient.Close()
			} else {
				panic(fmt.Errorf("close with error: %s", err).Error())
			}
			if c.perfClient != nil {
				c.perfClient.Close()
			}
		} else {
			if c.perfClient != nil {
				c.perfClient.Close()
			}
		}
		go func() {
			<-c.reconnectLoopDone
			<-c.streamLoopDone
			<-c.reportLoopDone
			c.report(c.state, true)
			c.qlog.Close()
			// flush qlog
			c.cancelQperfCtx()
		}()
	})
}

func (c *client) CloseWithError(err error) {
	c.close(err)
	<-c.qperfCtx.Done()
}

func (c *client) Close() {
	c.CloseWithError(nil)
}

func (c *client) TotalReport() common.Report {
	return c.state.TotalReport()
}
