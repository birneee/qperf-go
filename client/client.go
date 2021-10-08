package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/lucas-clemente/quic-go"
	"io"
	"net"
	"qperf-go/common"
	"time"
)

// Run client
func Run(addr net.UDPAddr) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"qperf"},
	}

	state := common.State{}

	conf := quic.Config{
		Tracer: common.StateTracer{
			State: &state,
		},
	}

	session, err := quic.DialAddr(addr.String(), tlsConf, &conf)
	if err != nil {
		panic(err)
	}

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

	go receive(stream)

	state.SetStartTime()

	fmt.Printf("time to first byte %s\n",
		humanize.SIWithDigits(state.GetOrWaitFirstByteTime().Sub(state.StartTime()).Seconds(), 2, "s"))

	for {
		if time.Now().Sub(state.GetOrWaitFirstByteTime()) > time.Duration(10*time.Second) {
			break
		}
		time.Sleep(1 * time.Second)
		report(&state)
	}

	stream.CancelRead(quic.StreamErrorCode(quic.NoError))

	err = session.CloseWithError(quic.ApplicationErrorCode(quic.NoError), "")
	if err != nil {
		panic(err)
	}

	reportTotal(&state)
}

func report(state *common.State) {
	receivedBytes, receivedPackets, delta := state.GetAndResetReport()
	fmt.Printf("second %.2f: %s, bytes received: %s, packets received: %d\n",
		time.Now().Sub(state.GetOrWaitFirstByteTime()).Seconds(),
		humanize.SIWithDigits(float64(receivedBytes)*8/delta.Seconds(), 2, "bit/s"),
		humanize.SI(float64(receivedBytes), "B"),
		receivedPackets)
}

func reportTotal(state *common.State) {
	receivedBytes, receivedPackets := state.Total()
	fmt.Printf("total: bytes received: %s, packets received: %d\n",
		humanize.SI(float64(receivedBytes), "B"),
		receivedPackets)
}

func receive(reader io.Reader) {
	dw := common.DiscardWriter{}
	_, err := io.Copy(dw, reader)
	if err != nil {
		//TODO differentiate errors from planed close
	}
}
