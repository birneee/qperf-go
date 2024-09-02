package perf

import "github.com/quic-go/quic-go"

// ALPN is from Section 2.1 in https://datatracker.ietf.org/doc/html/draft-banks-quic-performance-00
const ALPN = "perf"

const DefaultServerPort = 18080

const MaxResponseLength = ^uint64(0)
const MaxRequestLength = ^uint64(0)

const DeadlineExceededStreamErrorCode quic.StreamErrorCode = 1

type MessageType uint8

const (
	MessageTypeInvalid MessageType = iota
)
