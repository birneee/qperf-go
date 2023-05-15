package qlog_quic

// mostly copied from quic-go/qlog/types.go

import (
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
)

type owner uint8

const (
	ownerLocal owner = iota
	ownerRemote
)

func (o owner) String() string {
	switch o {
	case ownerLocal:
		return "local"
	case ownerRemote:
		return "remote"
	default:
		return "unknown owner"
	}
}

type streamType logging.StreamType

func (s streamType) String() string {
	switch logging.StreamType(s) {
	case logging.StreamTypeUni:
		return "unidirectional"
	case logging.StreamTypeBidi:
		return "bidirectional"
	default:
		return "unknown stream type"
	}
}

// category is the qlog event category.
type category = string

const (
	categoryConnectivity category = "connectivity"
	categoryTransport             = "transport"
	categorySecurity              = "security"
	categoryRecovery              = "recovery"
)

type versionNumber logging.VersionNumber

func (v versionNumber) String() string {
	return fmt.Sprintf("%x", uint32(v))
}

func (packetHeader) IsNil() bool { return false }

func encLevelToPacketNumberSpace(encLevel logging.EncryptionLevel) string {
	switch encLevel {
	case logging.EncryptionInitial:
		return "initial"
	case logging.EncryptionHandshake:
		return "handshake"
	case logging.Encryption0RTT, logging.Encryption1RTT:
		return "application_data"
	default:
		return "unknown encryption level"
	}
}

type keyType uint8

const (
	keyTypeServerInitial keyType = 1 + iota
	keyTypeClientInitial
	keyTypeServerHandshake
	keyTypeClientHandshake
	keyTypeServer0RTT
	keyTypeClient0RTT
	keyTypeServer1RTT
	keyTypeClient1RTT
)

func encLevelToKeyType(encLevel logging.EncryptionLevel, pers logging.Perspective) keyType {
	if pers == logging.PerspectiveServer {
		switch encLevel {
		case logging.EncryptionInitial:
			return keyTypeServerInitial
		case logging.EncryptionHandshake:
			return keyTypeServerHandshake
		case logging.Encryption0RTT:
			return keyTypeServer0RTT
		case logging.Encryption1RTT:
			return keyTypeServer1RTT
		default:
			return 0
		}
	}
	switch encLevel {
	case logging.EncryptionInitial:
		return keyTypeClientInitial
	case logging.EncryptionHandshake:
		return keyTypeClientHandshake
	case logging.Encryption0RTT:
		return keyTypeClient0RTT
	case logging.Encryption1RTT:
		return keyTypeClient1RTT
	default:
		return 0
	}
}

func (t keyType) String() string {
	switch t {
	case keyTypeServerInitial:
		return "server_initial_secret"
	case keyTypeClientInitial:
		return "client_initial_secret"
	case keyTypeServerHandshake:
		return "server_handshake_secret"
	case keyTypeClientHandshake:
		return "client_handshake_secret"
	case keyTypeServer0RTT:
		return "server_0rtt_secret"
	case keyTypeClient0RTT:
		return "client_0rtt_secret"
	case keyTypeServer1RTT:
		return "server_1rtt_secret"
	case keyTypeClient1RTT:
		return "client_1rtt_secret"
	default:
		return "unknown key type"
	}
}

type keyUpdateTrigger uint8

const (
	keyUpdateTLS keyUpdateTrigger = iota
	keyUpdateRemote
	keyUpdateLocal
)

func (t keyUpdateTrigger) String() string {
	switch t {
	case keyUpdateTLS:
		return "tls"
	case keyUpdateRemote:
		return "remote_update"
	case keyUpdateLocal:
		return "local_update"
	default:
		return "unknown key update trigger"
	}
}

type transportError uint64

func (e transportError) String() string {
	switch quic.TransportErrorCode(e) {
	case quic.NoError:
		return "no_error"
	case quic.InternalError:
		return "internal_error"
	case quic.ConnectionRefused:
		return "connection_refused"
	case quic.FlowControlError:
		return "flow_control_error"
	case quic.StreamLimitError:
		return "stream_limit_error"
	case quic.StreamStateError:
		return "stream_state_error"
	case quic.FinalSizeError:
		return "final_size_error"
	case quic.FrameEncodingError:
		return "frame_encoding_error"
	case quic.TransportParameterError:
		return "transport_parameter_error"
	case quic.ConnectionIDLimitError:
		return "connection_id_limit_error"
	case quic.ProtocolViolation:
		return "protocol_violation"
	case quic.InvalidToken:
		return "invalid_token"
	case quic.ApplicationErrorErrorCode:
		return "application_error"
	case quic.CryptoBufferExceeded:
		return "crypto_buffer_exceeded"
	case quic.KeyUpdateError:
		return "key_update_error"
	case quic.AEADLimitReached:
		return "aead_limit_reached"
	case quic.NoViablePathError:
		return "no_viable_path"
	default:
		return ""
	}
}

type packetType logging.PacketType

func (t packetType) String() string {
	switch logging.PacketType(t) {
	case logging.PacketTypeInitial:
		return "initial"
	case logging.PacketTypeHandshake:
		return "handshake"
	case logging.PacketTypeRetry:
		return "retry"
	case logging.PacketType0RTT:
		return "0RTT"
	case logging.PacketTypeVersionNegotiation:
		return "version_negotiation"
	case logging.PacketTypeStatelessReset:
		return "stateless_reset"
	case logging.PacketType1RTT:
		return "1RTT"
	case logging.PacketTypeNotDetermined:
		return ""
	default:
		return "unknown packet type"
	}
}

type packetLossReason logging.PacketLossReason

func (r packetLossReason) String() string {
	switch logging.PacketLossReason(r) {
	case logging.PacketLossReorderingThreshold:
		return "reordering_threshold"
	case logging.PacketLossTimeThreshold:
		return "time_threshold"
	default:
		return "unknown loss reason"
	}
}

type packetDropReason logging.PacketDropReason

func (r packetDropReason) String() string {
	switch logging.PacketDropReason(r) {
	case logging.PacketDropKeyUnavailable:
		return "key_unavailable"
	case logging.PacketDropUnknownConnectionID:
		return "unknown_connection_id"
	case logging.PacketDropHeaderParseError:
		return "header_parse_error"
	case logging.PacketDropPayloadDecryptError:
		return "payload_decrypt_error"
	case logging.PacketDropProtocolViolation:
		return "protocol_violation"
	case logging.PacketDropDOSPrevention:
		return "dos_prevention"
	case logging.PacketDropUnsupportedVersion:
		return "unsupported_version"
	case logging.PacketDropUnexpectedPacket:
		return "unexpected_packet"
	case logging.PacketDropUnexpectedSourceConnectionID:
		return "unexpected_source_connection_id"
	case logging.PacketDropUnexpectedVersion:
		return "unexpected_version"
	case logging.PacketDropDuplicate:
		return "duplicate"
	default:
		return "unknown packet drop reason"
	}
}

type timerType logging.TimerType

func (t timerType) String() string {
	switch logging.TimerType(t) {
	case logging.TimerTypeACK:
		return "ack"
	case logging.TimerTypePTO:
		return "pto"
	default:
		return "unknown timer type"
	}
}

type congestionState logging.CongestionState

func (s congestionState) String() string {
	switch logging.CongestionState(s) {
	case logging.CongestionStateSlowStart:
		return "slow_start"
	case logging.CongestionStateCongestionAvoidance:
		return "congestion_avoidance"
	case logging.CongestionStateRecovery:
		return "recovery"
	case logging.CongestionStateApplicationLimited:
		return "application_limited"
	default:
		return "unknown congestion state"
	}
}
