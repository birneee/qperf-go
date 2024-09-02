package perf_client

import (
	"context"
	errors2 "errors"
	"github.com/quic-go/quic-go"
	"net"
	"qperf-go/errors"
	"qperf-go/perf"
	"sync"
	"sync/atomic"
	"time"
)

type Client interface {
	Context() context.Context
	Request(requestLength uint64, responseLength uint64, responseDelay time.Duration) (RequestSendStream, ResponseReceiveStream, error)
	Close() error
	ReceivedBytes() uint64
	SentBytes() uint64
}

type client struct {
	conn                    quic.Connection
	config                  *Config
	closeOnce               sync.Once
	ctx                     context.Context
	cancelCtx               context.CancelCauseFunc
	receivedBytes           atomic.Uint64
	sentBytes               atomic.Uint64
	datagramReceiveLoopDone chan struct{}
}

func (c *client) Context() context.Context {
	return c.ctx
}

func DialAddr(remoteAddr string, conf *Config, early bool) (Client, error) {
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	t := quic.Transport{
		Conn:               udpConn,
		ConnectionIDLength: 4,
	}
	c := &client{
		config:                  conf.Populate(),
		datagramReceiveLoopDone: make(chan struct{}),
	}
	c.ctx, c.cancelCtx = context.WithCancelCause(context.Background())

	addr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return nil, err
	}

	if early {
		c.conn, err = t.DialEarly(c.ctx, addr, c.config.TlsConfig, c.config.QuicConfig)
	} else {
		c.conn, err = t.Dial(c.ctx, addr, c.config.TlsConfig, c.config.QuicConfig)
	}
	if err != nil {
		return nil, err
	}

	go func() {
		err := c.run()
		if err != nil {
			c.close(err)
		}
	}()

	return c, nil
}

func (c *client) run() error {
	go func() {
		err := c.runStreamAcceptLoop()
		if err != nil {
			c.close(err)
		}
	}()
	go func() {
		err := c.runDatagramReceiveLoop()
		if err != nil {
			c.close(err)
		}
	}()
	<-c.ctx.Done()
	return nil
}

func (c *client) acceptResponseReceiveStream() (ResponseReceiveStream, error) {
	_, err := c.conn.AcceptUniStream(context.Background())
	if err != nil {
		return nil, err
	}
	panic("implement me")
}

func (c *client) runStreamAcceptLoop() error {
	for {
		_, err := c.acceptResponseReceiveStream()
		if err != nil {
			return err
		}
		return nil
	}
}

func (c *client) Request(requestLength uint64, responseLength uint64, responseDelay time.Duration) (RequestSendStream, ResponseReceiveStream, error) {
	stream, err := c.conn.OpenStream()
	if err != nil {
		return nil, nil, err
	}
	requestStream := newRequestSendStream(stream, requestLength, responseLength, responseDelay, c)
	responseStream, err := newResponseReceiveStream(stream, c, responseLength)
	if err != nil {
		return nil, nil, err
	}
	return requestStream, responseStream, nil
}

// nil to close without error
func (c *client) close(err error) {
	c.closeOnce.Do(func() {
		if err != nil {
			_ = c.conn.CloseWithError(errors.InternalErrorCode, "internal error")
		} else {
			err = c.conn.CloseWithError(errors.NoError, "no error")
		}
		<-c.datagramReceiveLoopDone
		c.cancelCtx(err)
	})
}

func (c *client) Close() error {
	c.close(nil)
	cause := context.Cause(c.ctx)
	if !errors2.Is(cause, context.Canceled) {
		return cause
	}
	return nil
}

func (c *client) ReceivedBytes() uint64 {
	return c.receivedBytes.Load()
}

func (c *client) SentBytes() uint64 {
	return c.sentBytes.Load()
}

func (c *client) runDatagramReceiveLoop() error {
	defer close(c.datagramReceiveLoopDone)
	for {
		buf, err := c.conn.ReceiveDatagram(c.ctx)
		if err != nil {
			return err
		}
		messageType := perf.MessageType(buf[0])
		switch messageType {
		default:
			panic("unexpected message type")
		}
	}
}
