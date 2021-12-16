# QPERF-GO

A performance measurement tool for QUIC similar to iperf.
Uses https://github.com/birneee/quic-go

## Build
```bash
go build
```

## Setup
It is recommended to increase the maximum buffer size by running (See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details):

```bash
sysctl -w net.core.rmem_max=2500000
```

## Generate Self-signed certificate
```bash
openssl req -x509 -nodes -days 358000 -out server.crt -keyout server.key -config server.req # for server
openssl req -x509 -nodes -days 358000 -out proxy.crt -keyout proxy.key -config proxy.req # for proxy
```