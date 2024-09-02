# QPERF-GO

A performance measurement tool for QUIC ([RFC9000](https://datatracker.ietf.org/doc/html/rfc9000)) similar to iperf.
Uses https://github.com/quic-go/quic-go

## Features

- send and receive streams
- send and receive datagrams ([RFC9221](https://datatracker.ietf.org/doc/html/rfc9221)) (broken right now)
- qlog output ([draft-ietf-quic-qlog](https://datatracker.ietf.org/doc/draft-ietf-quic-qlog-main-schema/))
- 0-RTT handshakes
- CPU profiling

## Example
```bash
$ qperf client -a localhost -tsv -t 10s
{"qlog_format":"NDJSON","qlog_version":"draft-02","title":"qperf","code_version":"(devel)","trace":{"vantage_point":{"type":"client"},"common_fields":{"reference_time":1725288150744.6677,"time_format":"relative"}}}
{"time":1.362658,"name":"transport:connection_started","data":{"dst_cid":"b5803e506576ef35b7aef890"}}
{"time":2.567514,"name":"qperf:handshake_completed","data":{}}
{"time":2.708749,"name":"qperf:first_app_data_sent","data":{}}
{"time":3.03157,"name":"qperf:handshake_confirmed","data":{}}
{"time":3.122169,"name":"qperf:first_app_data_received","data":{}}
{"time":1000.35054,"name":"qperf:report","data":{"stream_mbps_received":7865.838,"stream_bytes_received":983565700,"period":1000.3416}}
{"time":2000.395328,"name":"qperf:report","data":{"stream_mbps_received":7966.526,"stream_bytes_received":995860348,"period":1000.04486}}
{"time":3000.423947,"name":"qperf:report","data":{"stream_mbps_received":7920.57,"stream_bytes_received":990099584,"period":1000.02856}}
{"time":4000.479466,"name":"qperf:report","data":{"stream_mbps_received":7893.4785,"stream_bytes_received":986739584,"period":1000.0556}}
{"time":5000.489054,"name":"qperf:report","data":{"stream_mbps_received":7984.5034,"stream_bytes_received":998072448,"period":1000.0096}}
{"time":6000.551982,"name":"qperf:report","data":{"stream_mbps_received":7968.698,"stream_bytes_received":996150016,"period":1000.0629}}
{"time":7000.584477,"name":"qperf:report","data":{"stream_mbps_received":7925.5615,"stream_bytes_received":990727424,"period":1000.0325}}
{"time":8000.604035,"name":"qperf:report","data":{"stream_mbps_received":7883.597,"stream_bytes_received":985469056,"period":1000.0196}}
{"time":9000.652448,"name":"qperf:report","data":{"stream_mbps_received":7903.3945,"stream_bytes_received":987972096,"period":1000.04834}}
{"time":10000.088981,"name":"transport:connection_closed","data":{"owner":"local","application_code":0,"reason":"no error"}}
{"time":10000.110586,"name":"qperf:total","data":{"stream_mbps_received":7920.8677,"stream_bytes_received":9901185920,"period":10000.102}}
```

## Requirements
- Go 1.23

## Build
```bash
go build
```

## Setup
It is recommended to increase the maximum buffer size by running (See https://github.com/quic-go/quic-go/wiki/UDP-Receive-Buffer-Size for details):

```bash
sysctl -w net.core.rmem_max=2500000
```

## Generate Self-signed certificate
```bash
openssl req -x509 -nodes -days 358000 -out server.crt -keyout server.key -config server.req
```