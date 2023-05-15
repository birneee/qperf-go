# QPERF-GO

A transport performance measurement tool for QUIC ([RFC9000](https://datatracker.ietf.org/doc/html/rfc9000)) similar to iperf.
Uses https://github.com/birneee/quic-go

## Features

- send and receive streams
- send and receive datagrams ([RFC9221](https://datatracker.ietf.org/doc/html/rfc9221))
- qlog output ([draft-ietf-quic-qlog](https://datatracker.ietf.org/doc/draft-ietf-quic-qlog-main-schema/))
- 0-RTT handshakes
- CPU profiling
- experimental XADS-QUIC extension (TBD)

## Example
```bash
$ qperf-go client -a localhost -xads
{"qlog_format":"NDJSON","qlog_version":"draft-02","title":"qperf","code_version":"(devel)","trace":{"vantage_point":{"type":"client"},"common_fields":{"reference_time":1684163451590.148,"time_format":"relative"}}}
{"time":1.095419,"name":"transport:connection_started","data":{"ip_version":"ipv6","src_ip":"::","src_port":46078,"dst_ip":"127.0.0.1","dst_port":18080,"src_cid":"(empty)","dst_cid":"bcfce24fc203637c9c034a"},"group_id":"bcfce24fc203637c9c034a","ODCID":"bcfce24fc203637c9c034a"}
{"time":3.466772,"name":"qperf:handshake_completed","data":{}}
{"time":3.47355,"name":"app:info","data":{"message":"use XADS-QUIC"}}
{"time":3.672632,"name":"qperf:handshake_confirmed","data":{}}
{"time":3.912037,"name":"qperf:first_app_data_received","data":{}}
{"time":1003.589264,"name":"qperf:report","data":{"stream_mbps_received":2473.7253,"stream_bytes_received":310322430,"period":1003.5793}}
{"time":2003.608461,"name":"qperf:report","data":{"stream_mbps_received":2541.7769,"stream_bytes_received":317728224,"period":1000.0192}}
{"time":3003.673975,"name":"qperf:report","data":{"stream_mbps_received":2501.291,"stream_bytes_received":312681798,"period":1000.06537}}
{"time":4003.692731,"name":"qperf:report","data":{"stream_mbps_received":2527.6216,"stream_bytes_received":315958698,"period":1000.0189}}
{"time":5003.730145,"name":"qperf:report","data":{"stream_mbps_received":2510.7983,"stream_bytes_received":313861482,"period":1000.03723}}
{"time":6003.740422,"name":"qperf:report","data":{"stream_mbps_received":2463.1548,"stream_bytes_received":307897524,"period":1000.0104}}
{"time":7003.75318,"name":"qperf:report","data":{"stream_mbps_received":2539.1724,"stream_bytes_received":317400534,"period":1000.01263}}
{"time":8003.797494,"name":"qperf:report","data":{"stream_mbps_received":2514.4502,"stream_bytes_received":314320248,"period":1000.04443}}
{"time":9003.802483,"name":"qperf:report","data":{"stream_mbps_received":2513.501,"stream_bytes_received":314189172,"period":1000.005}}
{"time":10003.811366,"name":"qperf:report","data":{"stream_mbps_received":2480.9849,"stream_bytes_received":310125816,"period":1000.00885}}
{"time":10003.828427,"name":"transport:connection_closed","data":{"owner":"local","application_code":0,"reason":"no error"},"group_id":"bcfce24fc203637c9c034a","ODCID":"bcfce24fc203637c9c034a"}
{"time":10003.85524,"name":"qperf:total","data":{"stream_mbps_received":2506.6792,"stream_bytes_received":3134554112,"period":10003.846}}
```

## Requirements
- Go 1.20

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