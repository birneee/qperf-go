package perf

import "github.com/quic-go/quic-go"

// ALPN is from Section 2.1 in https://datatracker.ietf.org/doc/html/draft-banks-quic-performance-00
const ALPN = "perf"

const DefaultServerPort = 18080

const MaxResponseLength = ^uint64(0)
const MaxDatagramResponseLength = 1197
const MaxRequestLength = ^uint64(0)
const MaxDatagramRequestLength = 1197

const MaxDatagramResponseNum = ^uint32(0)

const DeadlineExceededStreamErrorCode quic.StreamErrorCode = 1
