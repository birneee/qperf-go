package common

import (
	"context"
	"github.com/lucas-clemente/quic-go"
)

type SingleTokenStore struct {
	emptyContext       context.Context
	emptyContextCancel context.CancelFunc
	key                *string
	token              *quic.ClientToken
}

var _ quic.TokenStore = (*SingleTokenStore)(nil)

// Pop does not remove the token
func (s *SingleTokenStore) Pop(key string) (token *quic.ClientToken) {
	select {
	case <-s.emptyContext.Done():
		if key == *s.key {
			return s.token
		}
	default: // do not wait
	}
	return nil
}

func (s *SingleTokenStore) Put(key string, token *quic.ClientToken) {
	select {
	case <-s.emptyContext.Done():
		return // already set
	default:
		//TODO make thread safe
		s.key = &key
		s.token = token
		s.emptyContextCancel()
	}
}

func (s *SingleTokenStore) Await() (string, *quic.ClientToken) {
	<-s.emptyContext.Done()
	return *s.key, s.token
}
func NewSingleTokenStore() *SingleTokenStore {
	emptyContext, emptyContextCancel := context.WithCancel(context.Background())
	return &SingleTokenStore{
		emptyContext,
		emptyContextCancel,
		nil,
		nil,
	}
}
