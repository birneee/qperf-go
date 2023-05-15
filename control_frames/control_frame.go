package control_frames

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const HeaderLength = 3
const MaxFramePayloadLength = ^ControlFramePayloadLength(0)
const MaxFrameLength = HeaderLength + int(MaxFramePayloadLength)

type ControlFrameType uint8
type ControlFramePayloadLength uint16

const (
	FrameTypeStartSending          = 1
	FrameTypeData                  = 2
	FrameTypeStartSendingDatagrams = 3
)

type ControlFrame interface {
	Append([]byte) ([]byte, error)
}

type ControlFrameHeader struct {
	Type          ControlFrameType
	PayloadLength ControlFramePayloadLength
}

func (h ControlFrameHeader) Append(b []byte) []byte {
	b = append(b, byte(h.Type))
	b = append(b, 0, 0) // space for length
	binary.LittleEndian.PutUint16(b[1:], uint16(h.PayloadLength))
	return b
}

func ParseHeader(r *bytes.Reader) (*ControlFrameHeader, error) {
	frameType, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	var lenBuf [2]byte
	_, err = io.ReadFull(r, lenBuf[:])
	if err != nil {
		return nil, fmt.Errorf("failed to read frame length")
	}
	payloadLength := binary.LittleEndian.Uint16(lenBuf[:])
	return &ControlFrameHeader{
		Type:          ControlFrameType(frameType),
		PayloadLength: ControlFramePayloadLength(payloadLength),
	}, nil
}

func Parse(r *bytes.Reader) (ControlFrame, error) {
	header, err := ParseHeader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse header")
	}
	switch header.Type {
	case FrameTypeStartSending:
		return parseStartSendingFrame(r)
	case FrameTypeStartSendingDatagrams:
		return parseStartSendingDatagramsFrame(r)
	default:
		return nil, fmt.Errorf("unknown frame type")
	}
}
