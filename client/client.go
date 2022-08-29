package client

import (
	"crypto/tls"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/logging"
	"io"
	"net"
	"os"
	"os/signal"
	"qperf-go/common"
	"time"
)

// Run client.
func Run(addr net.UDPAddr, timeToFirstByteOnly bool, printRaw bool, createQLog bool, probeTime time.Duration, tlsServerCertFile string, initialReceiveWindow uint64, maxReceiveWindow uint64, use0RTT bool) {
	state := common.State{}

	tracers := make([]logging.Tracer, 0)

	tracers = append(tracers, common.StateTracer{
		State: &state,
	})

	if createQLog {
		tracers = append(tracers, common.NewQlogTrager("client"))
	}

	tracers = append(tracers, common.NewMigrationTracer(func(addr net.Addr) {
		fmt.Printf("migrated to %s\n", addr)
	}))

	if initialReceiveWindow > maxReceiveWindow {
		maxReceiveWindow = initialReceiveWindow
	}

	var clientSessionCache tls.ClientSessionCache
	if use0RTT {
		clientSessionCache = tls.NewLRUClientSessionCache(10)
	}

	var tokenStore quic.TokenStore
	if use0RTT {
		tokenStore = quic.NewLRUTokenStore(10, 10)
	}

	tlsConf := &tls.Config{
		RootCAs:            common.NewCertPoolWithCert(tlsServerCertFile),
		NextProtos:         []string{common.QperfALPN},
		ClientSessionCache: clientSessionCache,
	}

	conf := quic.Config{
		Tracer:                         logging.NewMultiplexedTracer(tracers...),
		InitialStreamReceiveWindow:     initialReceiveWindow,
		MaxStreamReceiveWindow:         maxReceiveWindow,
		InitialConnectionReceiveWindow: uint64(float64(initialReceiveWindow) * common.ConnectionFlowControlMultiplier),
		MaxConnectionReceiveWindow:     uint64(float64(maxReceiveWindow) * common.ConnectionFlowControlMultiplier),
		TokenStore:                     tokenStore,
	}

	if use0RTT {
		err := common.PingToGatherSessionTicketAndToken(addr.String(), tlsConf, &conf)
		if err != nil {
			panic(err)
		}
		fmt.Printf("stored session ticket and token\n")
	}

	state.SetStartTime()

	var connection quic.Connection
	if use0RTT {
		var err error
		connection, err = quic.DialAddrEarly(addr.String(), tlsConf, &conf)
		if err != nil {
			panic(err)
		}
	} else {
		var err error
		connection, err = quic.DialAddr(addr.String(), tlsConf, &conf)
		if err != nil {
			panic(err)
		}
	}

	state.SetEstablishmentTime()
	reportEstablishmentTime(&state, printRaw)

	// close gracefully on interrupt (CTRL+C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		_ = connection.CloseWithError(quic.ApplicationErrorCode(quic.NoError), "client_closed")
		os.Exit(0)
	}()

	stream, err := connection.OpenStream()
	if err != nil {
		panic(err)
	}

	// send some date to open stream
	_, err = stream.Write([]byte(common.QPerfStartSendingRequest))
	if err != nil {
		panic(err)
	}

	err = receiveFirstByte(stream, &state)
	if err != nil {
		panic(fmt.Errorf("failed to receive first byte: %w", err))
	}

	reportFirstByte(&state, printRaw)

	if !timeToFirstByteOnly {
		go receive(stream, &state)

		for {
			if time.Now().Sub(state.GetFirstByteTime()) > probeTime {
				break
			}
			time.Sleep(1 * time.Second)
			report(&state, printRaw)
		}
	}

	stream.CancelRead(quic.StreamErrorCode(quic.NoError))

	err = connection.CloseWithError(quic.ApplicationErrorCode(quic.NoError), "runtime_reached")
	if err != nil {
		panic(err)
	}

	reportTotal(&state, printRaw)
}

func reportEstablishmentTime(state *common.State, printRaw bool) {
	establishmentTime := state.EstablishmentTime().Sub(state.StartTime())
	if printRaw {
		fmt.Printf("connection establishment time: %f s\n",
			establishmentTime.Seconds())
	} else {
		fmt.Printf("connection establishment time: %s\n",
			humanize.SIWithDigits(establishmentTime.Seconds(), 2, "s"))
	}
}

func reportFirstByte(state *common.State, printRaw bool) {
	if printRaw {
		fmt.Printf("time to first byte: %f s\n",
			state.GetFirstByteTime().Sub(state.StartTime()).Seconds())
	} else {
		fmt.Printf("time to first byte: %s\n",
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
	buf := make([]byte, 65536)
	for {
		received, err := reader.Read(buf)
		state.AddReceivedBytes(uint64(received))
		if err != nil {
			//TODO differentiate errors from planed close
		}
	}
}
