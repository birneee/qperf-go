package common

import (
	"context"
	"crypto/tls"
	"github.com/marten-seemann/qtls-go1-19"
)

type SingleSessionCache struct {
	emptyContext       context.Context
	emptyContextCancel context.CancelFunc
	sessionKey         *string
	session            *tls.ClientSessionState
}

var _ qtls.ClientSessionCache = (*SingleSessionCache)(nil)

func (s *SingleSessionCache) Get(sessionKey string) (session *tls.ClientSessionState, ok bool) {
	select {
	case <-s.emptyContext.Done():
		if sessionKey == *s.sessionKey {
			return session, true
		}
	default: // do not wait
	}
	return nil, false
}

func (s *SingleSessionCache) Put(sessionKey string, cs *tls.ClientSessionState) {
	select {
	case <-s.emptyContext.Done():
		return // already set
	default:
		//TODO make thread safe
		s.sessionKey = &sessionKey
		s.session = cs
		s.emptyContextCancel()
	}
}

func (s *SingleSessionCache) Await() (string, *tls.ClientSessionState) {
	<-s.emptyContext.Done()
	return *s.sessionKey, s.session
}

func NewSingleSessionCache() *SingleSessionCache {
	emptyContext, emptyContextCancel := context.WithCancel(context.Background())
	return &SingleSessionCache{
		emptyContext,
		emptyContextCancel,
		nil,
		nil,
	}
}
