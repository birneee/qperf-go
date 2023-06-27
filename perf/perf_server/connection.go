package perf_server

import (
	"context"
	"encoding/binary"
	"github.com/quic-go/quic-go"
	"qperf-go/errors"
	"qperf-go/perf"
	"sync"
	"time"
)

type Connection interface {
	Context() context.Context
	TracingID() uint64
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
	datagramSendBuf     [perf.MaxDatagramResponseLength]byte
}

func NewConnection(quicConnection quic.EarlyConnection) Connection {
	c := &connection{
		quicConnection:        quicConnection,
		requestReceiveStreams: map[quic.StreamID]RequestReceiveStream{},
		responseSendStreams:   map[quic.StreamID]ResponseSendStream{},
	}
	go func() {
		err := c.runStreamLoop()
		if err != nil {
			c.close(err)
		}
	}()
	if quicConnection.ConnectionState().SupportsDatagrams {
		go func() {
			err := c.runDatagramLoop()
			if err != nil {
				c.close(err)
			}
		}()
	}
	return c
}

func (c *connection) newRequestReceiveStream(stream quic.Stream) (RequestReceiveStream, error) {
	reqStream, err := newRequestReceiveStream(stream, c)
	if err != nil {
		return nil, err
	}
	c.mutex.Lock()
	c.requestReceiveStreams[reqStream.StreamID()] = reqStream
	c.mutex.Unlock()
	go func() {
		<-reqStream.Context().Done()
		c.mutex.Lock()
		delete(c.requestReceiveStreams, reqStream.StreamID())
		c.mutex.Unlock()
	}()
	return reqStream, nil
}

func (c *connection) newResponseSendStream(stream quic.Stream, length uint64, delay time.Duration) (ResponseSendStream, error) {
	respStream := newResponseSendStream(stream, length, delay, c)
	c.mutex.Lock()
	c.responseSendStreams[respStream.StreamID()] = respStream
	c.mutex.Unlock()
	go func() {
		<-respStream.Context().Done()
		c.mutex.Lock()
		delete(c.responseSendStreams, respStream.StreamID())
		c.mutex.Unlock()
	}()
	return respStream, nil
}

func (c *connection) runStreamLoop() error {
	for {
		stream, err := c.quicConnection.AcceptStream(c.Context())
		if err != nil {
			return err
		}
		reqStream, err := c.newRequestReceiveStream(stream)
		if err != nil {
			return err
		}
		go func() {
			<-reqStream.Context().Done()
			if !reqStream.Success() {
				return // probably connection closed or deadline exceeded
			}
			if reqStream.ResponseLength() != 0 {
				_, err := c.newResponseSendStream(stream, reqStream.ResponseLength(), reqStream.ResponseDelay())
				if err != nil {
					c.close(err)
				}
			}
		}()
	}
}

func (c *connection) runDatagramLoop() error {
	for {
		msg, err := c.quicConnection.ReceiveMessage(c.Context())
		if err != nil {
			return err
		}
		responseNum := binary.LittleEndian.Uint32(msg[:4])
		responseLength := binary.LittleEndian.Uint32(msg[4:8])
		responseDelay := time.Duration(binary.LittleEndian.Uint32(msg[8:12])) * time.Millisecond
		if responseNum != 0 && responseLength != 0 {
			go func() {
				time.Sleep(responseDelay)
				for i := uint32(0); i < responseNum; i++ {
					err := c.quicConnection.SendMessage(c.datagramSendBuf[:responseLength])
					if err != nil {
						c.close(err)
						break
					}
				}
			}()
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

func (c *connection) TracingID() uint64 {
	return c.quicConnection.Context().Value(quic.ConnectionTracingKey).(uint64)
}

func (c *connection) QuicConn() quic.Connection {
	return c.quicConnection
}
