package control_frames

import (
	"bytes"
)

type StartSendingDatagramsFrame struct {
}

var _ ControlFrame = &StartSendingDatagramsFrame{}

func (s *StartSendingDatagramsFrame) Append(b []byte) ([]byte, error) {
	header := ControlFrameHeader{Type: FrameTypeStartSendingDatagrams, PayloadLength: ControlFramePayloadLength(0)}
	b = header.Append(b)
	return b, nil
}

func parseStartSendingDatagramsFrame(r *bytes.Reader) (*StartSendingDatagramsFrame, error) {
	return &StartSendingDatagramsFrame{}, nil
}
