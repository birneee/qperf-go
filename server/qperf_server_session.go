package server

import (
	"context"
	"github.com/quic-go/quic-go"
	"io"
	"qperf-go/common"
	"qperf-go/control_frames"
	"qperf-go/errors"
	"sync"
)

type qperfServerSession struct {
	quicConn     quic.Connection
	connectionID uint64
	// used to detect migration
	logger       common.Logger
	closeOnce    sync.Once
	streamSend   *common.AwaitableAtomicBool
	datagramSend *common.AwaitableAtomicBool
	config       *Config
}

// restoredQuicStreams is nil when not restored from hquic state
func newQperfConnection(quicConn quic.EarlyConnection, connectionID uint64, logger common.Logger, config *Config) (*qperfServerSession, error) {
	s := &qperfServerSession{
		quicConn:     quicConn,
		connectionID: connectionID,
		logger:       logger,
		streamSend:   common.NewAwaitableAtomicBool(false),
		datagramSend: common.NewAwaitableAtomicBool(false),
		config:       config,
	}

	go func() {
		quicControlStream, err := s.quicConn.AcceptStream(context.Background())
		if err != nil {
			s.close(err)
			return
		}
		controlStream := control_frames.NewControlFrameStream(quicControlStream)
		err = s.runCommandReceiver(controlStream)
		if err != nil {
			s.close(err)
			return
		}
	}()

	go func() {
		sendStream, err := s.quicConn.OpenUniStream()
		if err != nil {
			s.close(err)
		}
		err = s.runStreamSend(sendStream)
		if err != nil {
			s.close(err)
		}
	}()

	go func() {
		err := s.runDatagramSend()
		if err != nil {
			s.close(err)
		}
	}()

	go func() {
		err := s.runReceiveDatagram()
		if err != nil {
			s.close(err)
		}
	}()

	go func() {
		err := s.runAcceptReceiveStreams()
		if err != nil {
			s.close(err)
		}
	}()

	return s, nil
}

func (s *qperfServerSession) close(err error) {
	s.closeOnce.Do(func() {
		switch err := err.(type) {
		case *quic.ApplicationError:
			if err.ErrorCode == errors.NoError {
				// do nothing
			} else {
				s.logger.Errorf("close with error: %s", err)
			}
		default:
			s.logger.Errorf("close with error: %s", err)
		}
	})
}

func (s *qperfServerSession) runReceiveDatagram() error {
	for {
		_, err := s.quicConn.ReceiveMessage()
		if err != nil {
			return err
		}
	}
}

func (s *qperfServerSession) runCommandReceiver(fs control_frames.ControlFrameStream) error {
	for {
		f, err := fs.ReadFrame()
		if err != nil {
			return err
		}
		switch f.(type) {
		case *control_frames.StartSendingFrame:
			s.streamSend.Store(true)
		case *control_frames.StartSendingDatagramsFrame:
			s.datagramSend.Store(true)
		}
	}
}

func (s *qperfServerSession) runDatagramSend() error {
	var buf = make([]byte, 1197)
	//TODO calculate size from max_datagram_frame_size, max_udp_payload_size and path MTU
	for {
		s.datagramSend.AwaitTrue()
		err := s.quicConn.SendMessage(buf[:])
		if err != nil {
			return err
		}
	}
}

func (s *qperfServerSession) runStreamSend(stream quic.SendStream) error {
	var buf [65536]byte
	for {
		s.streamSend.AwaitTrue()
		_, err := stream.Write(buf[:])
		if err != nil {
			return err
		}
	}
}

func (s *qperfServerSession) runStreamReceive(stream quic.ReceiveStream) error {
	for {
		_, err := io.Copy(io.Discard, stream)
		if err != nil {
			return err
		}
	}
}

func (s *qperfServerSession) runAcceptReceiveStreams() error {
	for {
		receiveStream, err := s.quicConn.AcceptUniStream(context.Background())
		if err != nil {
			return err
		}

		go func() {
			err := s.runStreamReceive(receiveStream)
			if err != nil {
				s.close(err)
			}
		}()
	}
}
