package control_frames

import (
	"bytes"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/quicvarint"
)

type StartSendingFrame struct {
	StreamID quic.StreamID
}

var _ ControlFrame = &StartSendingFrame{}

func (s *StartSendingFrame) Append(b []byte) ([]byte, error) {
	header := ControlFrameHeader{Type: FrameTypeStartSending, PayloadLength: ControlFramePayloadLength(quicvarint.Len(uint64(s.StreamID)))}
	b = header.Append(b)
	b = quicvarint.Append(b, uint64(s.StreamID))
	return b, nil
}

func parseStartSendingFrame(r *bytes.Reader) (*StartSendingFrame, error) {
	sid, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}
	return &StartSendingFrame{
		StreamID: quic.StreamID(sid),
	}, nil
}
