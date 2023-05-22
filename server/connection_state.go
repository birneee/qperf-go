package server

import (
	"encoding/json"
	"github.com/quic-go/quic-go/handover"
)

type ConnectionState struct {
	QuicState     handover.State
	SendStream    bool
	SendDatagrams bool
}

func (s *ConnectionState) Serialize() ([]byte, error) {
	return json.Marshal(s)
}

func (s *ConnectionState) Parse(b []byte) (*ConnectionState, error) {
	if s == nil {
		s = &ConnectionState{}
	}
	err := json.Unmarshal(b, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}
