# QPERF-GO

A performance measurement tool for QUIC ([RFC9000](https://datatracker.ietf.org/doc/html/rfc9000)) similar to iperf.
Uses https://github.com/quic-go/quic-go

## Features

- send and receive streams
- send and receive datagrams ([RFC9221](https://datatracker.ietf.org/doc/html/rfc9221))
- qlog output ([draft-ietf-quic-qlog](https://datatracker.ietf.org/doc/draft-ietf-quic-qlog-main-schema/))
- 0-RTT handshakes
- CPU profiling

## Example
```bash
$ qperf-go client -a localhost
{"qlog_format":"NDJSON","qlog_version":"draft-02","title":"qperf","code_version":"(devel)","trace":{"vantage_point":{"type":"client"},"common_fields":{"reference_time":1684159160105.857,"time_format":"relative"}}}
{"time":1.005211,"name":"transport:connection_started","data":{"ip_version":"ipv6","src_ip":"::","src_port":38105,"dst_ip":"127.0.0.1","dst_port":18080,"src_cid":"(empty)","dst_cid":"7639ce1266656871c95b55d231"},"group_id":"7639ce1266656871c95b55d231","ODCID":"7639ce1266656871c95b55d231"}
{"time":2.493612,"name":"qperf:handshake_completed","data":{}}
{"time":2.777524,"name":"qperf:handshake_confirmed","data":{}}
{"time":2.883244,"name":"qperf:first_app_data_sent","data":{}}
{"time":3.167757,"name":"qperf:first_app_data_received","data":{}}
{"time":1000.134168,"name":"qperf:report","data":{"stream_mbps_received":6565.9297,"stream_bytes_received":820844056,"period":1000.1253}}
{"time":2000.1668,"name":"qperf:report","data":{"stream_mbps_received":6563.093,"stream_bytes_received":820413472,"period":1000.03265}}
{"time":3000.220085,"name":"qperf:report","data":{"stream_mbps_received":6536.362,"stream_bytes_received":817088748,"period":1000.05334}}
{"time":4000.241594,"name":"qperf:report","data":{"stream_mbps_received":6433.935,"stream_bytes_received":804259202,"period":1000.02155}}
{"time":5000.294673,"name":"qperf:report","data":{"stream_mbps_received":6540.979,"stream_bytes_received":817665882,"period":1000.0531}}
{"time":6000.363164,"name":"qperf:report","data":{"stream_mbps_received":6504.0977,"stream_bytes_received":813067816,"period":1000.06836}}
{"time":7000.437307,"name":"qperf:report","data":{"stream_mbps_received":6503.4683,"stream_bytes_received":812993820,"period":1000.0742}}
{"time":8000.463675,"name":"qperf:report","data":{"stream_mbps_received":6452.084,"stream_bytes_received":806531812,"period":1000.0265}}
{"time":9000.469631,"name":"qperf:report","data":{"stream_mbps_received":6519.3145,"stream_bytes_received":814919010,"period":1000.00586}}
{"time":10000.069076,"name":"transport:connection_closed","data":{"owner":"local","application_code":0,"reason":"no error"},"group_id":"7639ce1266656871c95b55d231","ODCID":"7639ce1266656871c95b55d231"}
{"time":10000.11039,"name":"qperf:total","data":{"stream_mbps_received":6509.1797,"stream_bytes_received":8136557954,"period":10000.102}}
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