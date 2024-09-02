package perf_server

import (
	"context"
	"github.com/quic-go/quic-go"
	"net"
	"sync"
)

type Server interface {
	Addr() net.Addr
	Close()
	Context() context.Context
}

type server struct {
	config    *Config
	listener  *quic.EarlyListener
	ctx       context.Context
	cancelCtx context.CancelFunc
	closeOnce sync.Once
	err       error
}

func (s *server) Context() context.Context {
	return s.ctx
}

func ListenAddr(addr string, config *Config) (Server, error) {
	s := &server{
		config: config.Populate(),
	}
	s.ctx, s.cancelCtx = context.WithCancel(context.Background())
	var err error
	s.listener, err = quic.ListenAddrEarly(addr, config.TlsConfig, config.QuicConfig)
	if err != nil {
		return nil, err
	}
	go func() {
		err := s.run()
		if err != nil {
			s.close(err)
		}
	}()
	return s, nil
}

func (s *server) run() error {
	for {
		quicConn, err := s.listener.Accept(s.ctx)
		if err != nil {
			s.close(err)
		}
		NewConnection(quicConn, s.config)
	}
}

func (s *server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *server) Close() {
	s.close(nil)
}

func (s *server) close(_ error) {
	s.closeOnce.Do(func() {
		err := s.listener.Close()
		s.err = err
		s.cancelCtx()
	})
}
