package common

import (
	"github.com/lucas-clemente/quic-go/logging"
	"net"
)

import (
	"context"
)

type eventTracer struct {
	logging.NullTracer
	handlers Handlers
}

type Handlers struct {
	StartedConnection func(odcid logging.ConnectionID, local, remote net.Addr, srcConnID, destConnID logging.ConnectionID)
	UpdatePath        func(odcid logging.ConnectionID, newRemote net.Addr)
	ClosedConnection  func(odcid logging.ConnectionID, err error)
}

func NewEventTracer(handlers Handlers) logging.Tracer {
	return &eventTracer{
		handlers: handlers,
	}
}

func (a eventTracer) TracerForConnection(ctx context.Context, p logging.Perspective, odcid logging.ConnectionID) logging.ConnectionTracer {
	return connectionEventTracer{
		odcid:   odcid,
		handers: a.handlers,
	}
}

type connectionEventTracer struct {
	logging.NullConnectionTracer
	odcid   logging.ConnectionID
	handers Handlers
}

func (c connectionEventTracer) StartedConnection(local, remote net.Addr, srcConnID, destConnID logging.ConnectionID) {
	if c.handers.StartedConnection != nil {
		c.handers.StartedConnection(c.odcid, local, remote, srcConnID, destConnID)
	}
}

func (c connectionEventTracer) UpdatedPath(newRemote net.Addr) {
	if c.handers.UpdatePath != nil {
		c.handers.UpdatePath(c.odcid, newRemote)
	}
}

func (c connectionEventTracer) ClosedConnection(err error) {
	if c.handers.ClosedConnection != nil {
		c.handers.ClosedConnection(c.odcid, err)
	}
}
