package qlog_quic

// mostly copied from quic-go/internal/wire/short_header.go

import (
	"github.com/quic-go/quic-go/logging"
)

func ShortHeaderLen(dest logging.ConnectionID, pnLen uint8) logging.ByteCount {
	return 1 + logging.ByteCount(dest.Len()) + logging.ByteCount(pnLen)
}
