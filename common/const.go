package common

const QperfALPN = "qperf"
const DefaultQperfServerPort = 18080
const DefaultProxyControlPort = 18081

// ConnectionFlowControlMultiplier is a copy of quic-go constant.
// determines how much larger the connection flow control windows needs to be relative to any stream's flow control window
// This is the value that Chromium is using
// do not modify, this should be the same as in quic-go!
const ConnectionFlowControlMultiplier = 1.5

// InitialCongestionWindow is a copy of quic-go constant.
// do not modify, this should be the same as in quic-go!
const InitialCongestionWindow = 32
