package control_frames

import (
	"bytes"
	"fmt"
	"github.com/quic-go/quic-go"
	"io"
)

type ControlFrameStream interface {
	// returns io.EOF when stream was closed
	ReadFrame() (ControlFrame, error)
	// skip data frames
	WriteFrame(ControlFrame) error
	WriteRaw(buf []byte) error
	ReadFrameInto(buf []byte) (int, ControlFrame, error)
}

type frameStream struct {
	quicStream quic.Stream
}

func (f *frameStream) ReadFrameInto(buf []byte) (int, ControlFrame, error) {
	var eof error
	_, err := io.ReadFull(f.quicStream, buf[:HeaderLength])
	if err != nil {
		return 0, nil, err
	}
	header, err := ParseHeader(bytes.NewReader(buf[:HeaderLength]))
	if err != nil {
		return 0, nil, err
	}
	if header.Type == FrameTypeData {
		_, err = io.CopyN(io.Discard, f.quicStream, int64(header.PayloadLength))
	} else {
		_, err = io.ReadFull(f.quicStream, buf[HeaderLength:int(header.PayloadLength)+HeaderLength])
	}
	if err != nil {
		if err == io.EOF {
			eof = err
		} else {
			return 0, nil, err
		}
	}
	frame, err := Parse(bytes.NewReader(buf[:int(header.PayloadLength)+HeaderLength]))
	if err != nil {
		return 0, nil, err
	}
	return int(header.PayloadLength) + HeaderLength, frame, eof
}

func (f *frameStream) ReadFrame() (ControlFrame, error) {
	var buf [MaxFrameLength]byte
	_, frame, err := f.ReadFrameInto(buf[:])
	return frame, err
}

func (f *frameStream) WriteFrame(frame ControlFrame) error {
	var buf []byte
	buf, err := frame.Append(buf)
	if err != nil {
		return err
	}
	return f.WriteRaw(buf)
}

func (f *frameStream) WriteRaw(buf []byte) error {
	n, err := f.quicStream.Write(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("failed to write all bytes")
	}
	return nil
}

func NewControlFrameStream(quicStream quic.Stream) ControlFrameStream {
	return &frameStream{
		quicStream: quicStream,
	}
}
