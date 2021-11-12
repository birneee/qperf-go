package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/logging"
	"github.com/lucas-clemente/quic-go/qlog"
	"io"
	"net"
	"os"
	"qperf-go/common"
	"time"
)

// Run server
func Run(addr net.UDPAddr, createQLog bool, migrateAfter time.Duration) {

	state := common.State{}

	multiTracer := common.MultiTracer{}

	multiTracer.Tracers = append(multiTracer.Tracers, common.StateTracer{
		State: &state,
	})

	if createQLog {
		multiTracer.Tracers = append(multiTracer.Tracers, qlog.NewTracer(func(p logging.Perspective, connectionID []byte) io.WriteCloser {
			filename := fmt.Sprintf("server_%x.qlog", connectionID)
			f, err := os.Create(filename)
			if err != nil {
				panic(err)
			}
			return common.NewBufferedWriteCloser(bufio.NewWriter(f), f)
		}))
	}

	conf := quic.Config{
		Tracer: multiTracer,
	}

	//TODO make CLI option
	tlsCert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		panic(err)
	}

	tlsConf := tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"qperf"},
	}

	listener, err := quic.ListenAddr(addr.String(), &tlsConf, &conf)
	if err != nil {
		panic(err)
	}

	fmt.Printf("starting server with %d, port %d, cc cubic, iw %d\n", os.Getpid(), addr.Port, conf.InitialConnectionReceiveWindow)

	// migrate
	if migrateAfter.Nanoseconds() != 0 {
		go func() {
			time.Sleep(migrateAfter)
			addr, err := listener.Migrate()
			if err != nil {
				panic(err)
			}
			fmt.Printf("migrated to %s\n", addr.String())
		}()
	}

	var nextSessionId uint64 = 0

	for {
		session, err := listener.Accept(context.Background())
		if err != nil {
			panic(err)
		}
		fmt.Printf("[session %d] open session\n", nextSessionId)
		go handleSession(session, nextSessionId)
		nextSessionId += 1
	}
}

func handleSession(session quic.Session, sessionId uint64) {
	for {
		stream, err := session.AcceptStream(context.Background())
		if err != nil {
			fmt.Printf("[session %d] %s\n", sessionId, err)
			return
		}
		fmt.Printf("[session %d][stream %d] open stream\n", sessionId, stream.StreamID())
		go sendData(stream, sessionId)
	}
}

func sendData(stream quic.SendStream, sessionId uint64) {
	buf := make([]byte, 1024)
	for {
		_, err := stream.Write(buf)
		if err != nil {
			fmt.Printf("[session %d][stream %d] %s\n", sessionId, stream.StreamID(), err)
			return
		}
	}
}
