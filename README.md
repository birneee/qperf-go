# Setup
It is recommended to increase the maximum buffer size by running (See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details):

```
sysctl -w net.core.rmem_max=2500000
```

## Generate Self-signed certificate
```
openssl req -x509 -nodes -out server.crt -keyout server.key -config server.req
```