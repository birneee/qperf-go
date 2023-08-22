package qlog_quic

import (
	"fmt"
	"qperf-go/common/qlog"
	"reflect"
)

type config struct {
	LogTransportConnectionStarted     bool
	LogTransportVersionInformation    bool
	LogTransportConnectionClosed      bool
	LogTransportPacketReceived        bool
	LogTransportPacketSent            bool
	LogRecoveryMetricsUpdated         bool
	LogRecoveryLossTimerUpdated       bool
	LogTransportPacketBuffered        bool
	LogTransportPacketDropped         bool
	LogRecoveryPacketLost             bool
	LogSecurityKeyUpdated             bool
	LogSecurityKeyDiscarded           bool
	LogTransportParametersRestored    bool
	LogTransportParametersSet         bool
	LogRecoveryCongestionStateUpdated bool
}

func (c *config) ApplyConf(qlogConfig qlog.Config) {
	c.SetIncludeAll(!qlogConfig.ExcludeEventsByDefault)
	for name, include := range qlogConfig.IncludedEvents {
		c.SetIncludeByName(fmt.Sprintf("%s:%s", name.Category, name.Name), include)
	}
}

func (c *config) SetIncludeAll(include bool) {
	v := reflect.ValueOf(c).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		f.SetBool(include)
	}
}

// SetIncludeByName
// does nothing if name does not match
func (c *config) SetIncludeByName(name string, include bool) {
	switch name {
	case "transport:connection_started":
		c.LogTransportConnectionStarted = include
	case "transport:version_information":
		c.LogTransportVersionInformation = include
	case "transport:connection_closed":
		c.LogTransportConnectionClosed = include
	case "transport:packet_sent":
		c.LogTransportPacketSent = include
	case "transport:packet_received":
		c.LogTransportPacketReceived = include
	case "transport:packet_buffered":
		c.LogTransportPacketBuffered = include
	case "transport:packet_dropped":
		c.LogTransportPacketDropped = include
	case "recovery:metrics_updated":
		c.LogRecoveryMetricsUpdated = include
	case "recovery:packet_lost":
		c.LogRecoveryPacketLost = include
	case "security:key_updated":
		c.LogSecurityKeyUpdated = include
	case "security:key_discarded":
		c.LogSecurityKeyDiscarded = include
	case "transport:parameters_restored":
		c.LogTransportParametersRestored = include
	case "transport:parameters_set":
		c.LogTransportParametersSet = include
	case "recovery:loss_timer_updated":
		c.LogRecoveryLossTimerUpdated = include
	case "recovery:congestion_state_updated":
		c.LogRecoveryCongestionStateUpdated = include
	}
}
