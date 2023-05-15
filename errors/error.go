package errors

import "github.com/quic-go/quic-go"

const (
	NoError           = quic.ApplicationErrorCode(0)
	InternalErrorCode = quic.ApplicationErrorCode(1)
)
