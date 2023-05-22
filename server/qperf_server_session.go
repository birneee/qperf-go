package server

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
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

func restoreQperfConnection(state *ConnectionState, listener *quic.EarlyListener, connectionID uint64, logger common.Logger, tracer func(context.Context, logging.Perspective, logging.ConnectionID) logging.ConnectionTracer, config *Config) (*qperfServerSession, error) {
	restoredQuicConn, restoredQuicStreams, err := quic.Restore(state.QuicState, &quic.ConnectionRestoreConfig{
		Perspective: logging.PerspectiveServer,
		Listener:    listener,
		QuicConf: &quic.Config{
			EnableDatagrams: true,
			Tracer:          tracer,
		},
	})
	if err != nil {
		return nil, err
	}
	s := &qperfServerSession{
		quicConn:     restoredQuicConn,
		connectionID: connectionID,
		logger:       logger,
		streamSend:   common.NewAwaitableAtomicBool(state.SendStream),
		datagramSend: common.NewAwaitableAtomicBool(state.SendDatagrams),
		config:       config,
	}
	quicControlStream, ok := restoredQuicStreams.BidiStreams[0]
	if !ok {
		return nil, fmt.Errorf("no control stream")
	}
	controlStream := control_frames.NewControlFrameStream(quicControlStream)

	sendStream, ok := restoredQuicStreams.SendStreams[3]
	if !ok {
		return nil, fmt.Errorf("no send stream")
	}

	go func() {
		err := s.runCommandReceiver(controlStream)
		if err != nil {
			s.close(err)
			return
		}
	}()

	go func() {
		err := s.runStreamSend(sendStream)
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

	for _, receiveStream := range restoredQuicStreams.ReceiveStreams {
		receiveStream := receiveStream
		go func() {
			err := s.runStreamReceive(receiveStream)
			if err != nil {
				s.close(err)
			}
		}()
	}

	go func() {
		err := s.runAcceptReceiveStreams()
		if err != nil {
			s.close(err)
		}
	}()

	return s, nil
}

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

func (s *qperfServerSession) Handover() (*ConnectionState, error) {
	resp := s.quicConn.Handover(true, &quic.ConnectionStateStoreConf{
		IgnoreCurrentPath:            true,
		IncludePendingOutgoingFrames: s.config.StateIncludesPendingStreamFrames,
		IncludePendingIncomingFrames: s.config.StateIncludesPendingStreamFrames,
		IncludeCongestionState:       s.config.StateIncludesCongestionState,
	})
	quicState := resp.State
	err := resp.Error
	if err != nil {
		return nil, err
	}
	qperfState := &ConnectionState{
		QuicState:     quicState,
		SendStream:    s.streamSend.Load(),
		SendDatagrams: s.datagramSend.Load(),
	}
	return qperfState, err
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
