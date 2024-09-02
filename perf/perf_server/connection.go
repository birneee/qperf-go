package perf_server

import (
	"context"
	"github.com/quic-go/quic-go"
	"qperf-go/errors"
	"sync"
	"time"
)

type Connection interface {
	Context() context.Context
	TracingID() quic.ConnectionTracingID
	// Close connection without error
	Close()
	QuicConn() quic.Connection
}

type connection struct {
	quicConnection quic.Connection
	closeOnce      sync.Once
	// only set within closeOnce
	err   error
	mutex sync.Mutex
	// only access while holding mutex
	requestReceiveStreams map[quic.StreamID]RequestReceiveStream
	// only access while holding mutex
	responseSendStreams map[quic.StreamID]ResponseSendStream
	config              *Config
}

func NewConnection(quicConnection quic.EarlyConnection, config *Config) Connection {
	c := &connection{
		quicConnection:        quicConnection,
		requestReceiveStreams: map[quic.StreamID]RequestReceiveStream{},
		responseSendStreams:   map[quic.StreamID]ResponseSendStream{},
		config:                config,
	}
	go func() {
		err := c.run()
		if err != nil {
			c.close(err)
		}
	}()
	return c
}

func (c *connection) newRequestReceiveStream(stream quic.Stream) (RequestReceiveStream, error) {
	reqStream, err := newRequestReceiveStream(stream, c)
	if err != nil {
		return nil, err
	}
	return reqStream, c.handleRequestReceiveStream(reqStream, stream)
}

func (c *connection) handleRequestReceiveStream(reqStream RequestReceiveStream, quicStream quic.Stream) error {
	c.mutex.Lock()
	c.requestReceiveStreams[reqStream.StreamID()] = reqStream
	c.mutex.Unlock()
	go func() {
		select {
		case <-reqStream.Context().Done():
			// remove if failed, otherwise request is removed after response
			if !reqStream.Success() || reqStream.ResponseLength() == 0 {
				c.mutex.Lock()
				delete(c.requestReceiveStreams, reqStream.StreamID())
				c.mutex.Unlock()
				return
			}
			_, err := c.newResponseSendStream(quicStream, reqStream.ResponseLength(), reqStream.ResponseDelay())
			if err != nil {
				c.close(err)
			}
		}
	}()
	return nil
}

func (c *connection) newResponseSendStream(stream quic.Stream, length uint64, delay time.Duration) (ResponseSendStream, error) {
	respStream := newResponseSendStream(stream, length, delay, c)
	c.mutex.Lock()
	c.responseSendStreams[respStream.StreamID()] = respStream
	c.mutex.Unlock()
	go func() {
		select {
		case <-respStream.Context().Done():
			c.mutex.Lock()
			delete(c.requestReceiveStreams, respStream.StreamID())
			delete(c.responseSendStreams, respStream.StreamID())
			c.mutex.Unlock()
		}
	}()
	return respStream, nil
}

func (c *connection) run() error {
	for {
		stream, err := c.quicConnection.AcceptStream(c.Context())
		if err != nil {
			return err
		}
		_, err = c.newRequestReceiveStream(stream)
		if err != nil {
			return err
		}
	}
}

func (c *connection) close(err error) {
	c.closeOnce.Do(func() {
		if err != nil {
			err := c.quicConnection.CloseWithError(errors.InternalErrorCode, "internal error")
			c.err = err
		} else {
			err := c.quicConnection.CloseWithError(errors.NoError, "no error")
			c.err = err
		}
	})
}

func (c *connection) Close() {
	c.close(nil)
}

func (c *connection) Context() context.Context {
	return c.quicConnection.Context()
}

func (c *connection) TracingID() quic.ConnectionTracingID {
	return c.quicConnection.Context().Value(quic.ConnectionTracingKey).(quic.ConnectionTracingID)
}

func (c *connection) QuicConn() quic.Connection {
	return c.quicConnection
}
