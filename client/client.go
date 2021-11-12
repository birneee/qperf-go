package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/logging"
	"github.com/lucas-clemente/quic-go/qlog"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"qperf-go/common"
	"time"
)

// Run client
func Run(addr net.UDPAddr, timeToFirstByteOnly bool, printRaw bool, createQLog bool, migrateAfter time.Duration) {
	certPool := x509.NewCertPool()

	//TODO add CLI options
	caCertRaw, err := ioutil.ReadFile("server.crt")
	if err != nil {
		panic(err)
	}

	ok := certPool.AppendCertsFromPEM(caCertRaw)
	if !ok {
		panic("failed to add certificate to pool")
	}

	tlsConf := &tls.Config{
		RootCAs:    certPool,
		NextProtos: []string{"qperf"},
	}

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

	session, err := quic.DialAddr(addr.String(), tlsConf, &conf)
	if err != nil {
		panic(err)
	}

	//TODO remove handover test
	//println("connected")
	//time.Sleep(time.Second)
	//session, err = session.Clone()
	//if err != nil {
	//	panic(err)
	//}
	//println("cloned")

	// migrate
	if migrateAfter.Nanoseconds() != 0 {
		go func() {
			time.Sleep(migrateAfter)
			addr, err := session.Migrate()
			if err != nil {
				panic(err)
			}
			fmt.Printf("migrated to %s\n", addr.String())
		}()
	}

	state.SetStartTime()

	// close gracefully on interrupt (CTRL+C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		err = session.CloseWithError(quic.ApplicationErrorCode(quic.NoError), "client_closed")
		os.Exit(0)
	}()

	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		panic(err)
	}

	// send some date to open stream
	_, err = stream.Write([]byte("qperf start sending"))
	if err != nil {
		panic(err)
	}

	stream.CancelWrite(quic.StreamErrorCode(quic.NoError))

	err = receiveFirstByte(stream, &state)
	if err != nil {
		panic(err)
	}

	reportFirstByte(&state, printRaw)

	if !timeToFirstByteOnly {
		go receive(stream, &state)

		for {
			if time.Now().Sub(state.GetFirstByteTime()) > 10*time.Second {
				break
			}
			time.Sleep(1 * time.Second)
			report(&state, printRaw)
		}
	}

	stream.CancelRead(quic.StreamErrorCode(quic.NoError))

	err = session.CloseWithError(quic.ApplicationErrorCode(quic.NoError), "runtime_reached")
	if err != nil {
		panic(err)
	}

	reportTotal(&state, printRaw)
}

func reportFirstByte(state *common.State, printRaw bool) {
	if printRaw {
		fmt.Printf("time to first byte %f s\n",
			state.GetFirstByteTime().Sub(state.StartTime()).Seconds())
	} else {
		fmt.Printf("time to first byte %s\n",
			humanize.SIWithDigits(state.GetFirstByteTime().Sub(state.StartTime()).Seconds(), 2, "s"))
	}
}

func report(state *common.State, printRaw bool) {
	receivedBytes, receivedPackets, delta := state.GetAndResetReport()
	if printRaw {
		fmt.Printf("second %f: %f bit/s, bytes received: %d B, packets received: %d\n",
			time.Now().Sub(state.GetFirstByteTime()).Seconds(),
			float64(receivedBytes)*8/delta.Seconds(),
			receivedBytes,
			receivedPackets)
	} else {
		fmt.Printf("second %.2f: %s, bytes received: %s, packets received: %d\n",
			time.Now().Sub(state.GetFirstByteTime()).Seconds(),
			humanize.SIWithDigits(float64(receivedBytes)*8/delta.Seconds(), 2, "bit/s"),
			humanize.SI(float64(receivedBytes), "B"),
			receivedPackets)
	}
}

func reportTotal(state *common.State, printRaw bool) {
	receivedBytes, receivedPackets := state.Total()
	if printRaw {
		fmt.Printf("total: bytes received: %d B, packets received: %d\n",
			receivedBytes,
			receivedPackets)
	} else {
		fmt.Printf("total: bytes received: %s, packets received: %d\n",
			humanize.SI(float64(receivedBytes), "B"),
			receivedPackets)
	}
}

func receiveFirstByte(stream quic.ReceiveStream, state *common.State) error {
	buf := make([]byte, 1)
	for {
		received, err := stream.Read(buf)
		if err != nil {
			return err
		}
		if received != 0 {
			state.AddReceivedBytes(uint64(received))
			return nil
		}
	}
}

func receive(reader io.Reader, state *common.State) {
	buf := make([]byte, 1024)
	for {
		received, err := reader.Read(buf)
		state.AddReceivedBytes(uint64(received))
		if err != nil {
			//TODO differentiate errors from planed close
		}
	}
}