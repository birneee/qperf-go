package common

var QlogLevelDefaultEvents = QlogLevelInfoEvents

var QlogLevelInfoEvents = map[string]bool{
	"transport:connection_started":  true,
	"qperf:handshake_completed":     true,
	"qperf:handshake_confirmed":     true,
	"qperf:first_app_data_received": true,
	"qperf:report":                  true,
	"transport:path_updated":        true,
	"qperf:total":                   true,
	"transport:connection_closed":   true,
	"app:info":                      true,
	"app:error":                     true,
}
