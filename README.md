# QPERF-GO

A performance measurement tool for QUIC ([RFC9000](https://datatracker.ietf.org/doc/html/rfc9000)) similar to iperf.
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
{"time":2.26899,"name":"app:info","data":{"message":"use XADS-QUIC"}}
{"time":2.21681,"name":"qperf:handshake_completed","data":{}}
{"time":2.523389,"name":"qperf:handshake_confirmed","data":{}}
{"time":2.656643,"name":"qperf:first_app_data_sent","data":{}}
{"time":2.886065,"name":"qperf:first_app_data_received","data":{}}
{"time":1000.3204,"name":"qperf:report","data":{"stream_mbps_received":4017.4238,"stream_bytes_received":502333440,"period":1000.3096}}
{"time":2000.328104,"name":"qperf:report","data":{"stream_mbps_received":4087.5806,"stream_bytes_received":510951424,"period":1000.0075}}
{"time":3000.352802,"name":"qperf:report","data":{"stream_mbps_received":4092.884,"stream_bytes_received":511623168,"period":1000.02484}}
{"time":4000.45348,"name":"qperf:report","data":{"stream_mbps_received":4067.5408,"stream_bytes_received":508493824,"period":1000.1007}}
{"time":5000.467069,"name":"qperf:report","data":{"stream_mbps_received":4075.8906,"stream_bytes_received":509493248,"period":1000.0136}}
{"time":6000.483748,"name":"qperf:report","data":{"stream_mbps_received":4142.724,"stream_bytes_received":517849088,"period":1000.01654}}
{"time":7000.565416,"name":"qperf:report","data":{"stream_mbps_received":4068.0115,"stream_bytes_received":508542976,"period":1000.0816}}
{"time":8000.593553,"name":"qperf:report","data":{"stream_mbps_received":4063.904,"stream_bytes_received":508002304,"period":1000.02814}}
{"time":9000.638564,"name":"qperf:report","data":{"stream_mbps_received":4089.5242,"stream_bytes_received":511213568,"period":1000.0451}}
{"time":10000.155606,"name":"transport:connection_closed","data":{"owner":"local","application_code":0,"reason":"no error"},"group_id":"bcfce24fc203637c9c034a","ODCID":"bcfce24fc203637c9c034a"}
{"time":10000.18806,"name":"qperf:total","data":{"stream_mbps_received":4073.2524,"stream_bytes_received":5091655680,"period":10000.178}}
```

## Requirements
- Go 1.20 (1.21 not yet supported)

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