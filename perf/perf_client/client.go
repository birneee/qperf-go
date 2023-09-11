package perf_client

import (
	"context"
	"encoding/binary"
	"errors"
	"github.com/quic-go/quic-go"
	"qperf-go/common"
	errors2 "qperf-go/errors"
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
	DatagramRequest(requestLength uint64, responseNum uint32, responseLength uint32, responseDelay time.Duration) error
	ExtraApplicationDataSecurity() bool
	ConnectionState() quic.ConnectionState
}

type client struct {
	conn          quic.Connection
	config        *Config
	closeOnce     sync.Once
	ctx           context.Context
	cancelCtx     context.CancelFunc
	err           error
	receivedBytes atomic.Uint64
	sentBytes     atomic.Uint64
	datagramBuf   [perf.MaxDatagramRequestLength]byte
}

func (c *client) Context() context.Context {
	return c.ctx
}

func DialAddr(remoteAddr string, conf *Config) (Client, error) {
	c := &client{
		config: conf.Populate(),
	}
	c.ctx, c.cancelCtx = context.WithCancel(context.Background())

	var err error
	c.conn, err = quic.DialAddr(c.ctx, remoteAddr, c.config.TlsConfig, c.config.QuicConfig)
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

func DialEarlyAddr(remoteAddr string, conf *Config) (Client, error) {
	c := &client{
		config: conf.Populate(),
	}
	c.ctx, c.cancelCtx = context.WithCancel(context.Background())

	var err error
	c.conn, err = quic.DialAddrEarly(c.ctx, remoteAddr, c.config.TlsConfig, c.config.QuicConfig)
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

	<-c.ctx.Done()
	return nil
}

func (c *client) acceptResponseReceiveStream() (ResponseReceiveStream, error) {
	quicStream, err := c.conn.AcceptUniStream(context.Background())
	if err != nil {
		return nil, err
	}
	return newResponseReceiveStream(quicStream, c)
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
	go func() {
		ctx := requestStream.Context()
		<-ctx.Done()
		err := ctx.Err()
		var streamErr *quic.StreamError
		switch {
		case errors.Is(err, context.Canceled):
		case errors.As(streamErr, &err):
			switch streamErr.ErrorCode {
			case perf.DeadlineExceededStreamErrorCode:
			default:
				c.close(err)
			}
		default:
			c.close(err)
		}
	}()
	var responseStream ResponseReceiveStream = nil
	if responseLength != 0 {
		responseStream, err = newResponseReceiveStream(stream, c)
		if err != nil {
			return nil, nil, err
		}
	}
	return requestStream, responseStream, nil
}

// nil to close without error
func (c *client) close(err error) {
	c.closeOnce.Do(func() {
		if err != nil {
			err := c.conn.CloseWithError(errors2.InternalErrorCode, "internal error")
			c.err = err
		} else {
			err := c.conn.CloseWithError(errors2.NoError, "no error")
			c.err = err
		}
		c.cancelCtx()
	})
}

func (c *client) Close() error {
	c.close(nil)
	return c.err
}

func (c *client) ReceivedBytes() uint64 {
	return c.receivedBytes.Load()
}

func (c *client) SentBytes() uint64 {
	return c.sentBytes.Load()
}

func (c *client) DatagramRequest(requestLength uint64, responseNum uint32, responseLength uint32, responseDelay time.Duration) error {
	binary.LittleEndian.PutUint32(c.datagramBuf[:], responseNum)
	binary.LittleEndian.PutUint32(c.datagramBuf[4:], responseLength)
	binary.LittleEndian.PutUint32(c.datagramBuf[8:], uint32(responseDelay.Milliseconds()))
	err := c.conn.SendMessage(c.datagramBuf[:common.Max(12, requestLength)])
	if err != nil {
		return err
	}
	return nil
}

func (c *client) ExtraApplicationDataSecurity() bool {
	return c.conn.ExtraApplicationDataSecurity()
}

func (c *client) ConnectionState() quic.ConnectionState {
	return c.conn.ConnectionState()
}
