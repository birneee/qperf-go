package common

import "github.com/lucas-clemente/quic-go"

// TODO IANA registration
const QperfALPN = "qperf"

const DefaultQperfServerPort = 18080

const RuntimeReachedErrorCode = quic.ApplicationErrorCode(0)
