package server

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/logging"
	"github.com/lucas-clemente/quic-go/qlog"
	"io"
	"math/big"
	"net"
	"os"
	"qperf-go/common"
	"time"
)

// Run server
//
// The sync.WaitGroup may be nil
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

	listener, err := quic.ListenAddr(addr.String(), generateTLSConfig(), &conf)
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

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"qperf"},
	}
}
