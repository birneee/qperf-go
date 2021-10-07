package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/lucas-clemente/quic-go"
)

// Run client
func Run(addr string) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"qperf"},
	}

	session, err := quic.DialAddr(addr, tlsConf, nil)
	if err != nil {
		panic(err)
	}

	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		panic(err)
	}

	buf := []byte("hello")

	_, err = stream.Write(buf)
	if err != nil {
		panic(err)
	}
	fmt.Printf("client sent: %s\n", buf)

	err = stream.Close()
	if err != nil {
		panic(err)
	}
}
