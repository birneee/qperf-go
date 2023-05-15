# QPERF-GO

A performance measurement tool for QUIC similar to iperf.
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
{"time":2.588341,"name":"qperf:handshake_completed","data":{}}
{"time":2.817386,"name":"qperf:handshake_confirmed","data":{}}
{"time":2.887085,"name":"qperf:first_app_data_received","data":{}}
{"time":1002.693537,"name":"qperf:report","data":{"stream_mbps_received":2886.4092,"stream_bytes_received":361769760,"period":1002.6846}}
{"time":2002.716831,"name":"qperf:report","data":{"stream_mbps_received":2910.8682,"stream_bytes_received":363866976,"period":1000.02325}}
{"time":3002.724152,"name":"qperf:report","data":{"stream_mbps_received":2901.4773,"stream_bytes_received":362687292,"period":1000.00726}}
{"time":4002.760003,"name":"qperf:report","data":{"stream_mbps_received":2850.0146,"stream_bytes_received":356264568,"period":1000.03577}}
{"time":5002.845086,"name":"qperf:report","data":{"stream_mbps_received":2898.1057,"stream_bytes_received":362294064,"period":1000.08527}}
{"time":6002.898535,"name":"qperf:report","data":{"stream_mbps_received":2869.8872,"stream_bytes_received":358755012,"period":1000.0533}}
{"time":7002.92101,"name":"qperf:report","data":{"stream_mbps_received":2908.2488,"stream_bytes_received":363539286,"period":1000.0226}}
{"time":8002.959911,"name":"qperf:report","data":{"stream_mbps_received":2898.24,"stream_bytes_received":362294064,"period":1000.0389}}
{"time":9003.034749,"name":"qperf:report","data":{"stream_mbps_received":2872.9707,"stream_bytes_received":359148240,"period":1000.0749}}
{"time":10003.04349,"name":"qperf:report","data":{"stream_mbps_received":2907.2405,"stream_bytes_received":363408210,"period":1000.00867}}
{"time":10003.063587,"name":"transport:connection_closed","data":{"owner":"local","application_code":0,"reason":"no error"},"group_id":"7639ce1266656871c95b55d231","ODCID":"7639ce1266656871c95b55d231"}
{"time":10003.096202,"name":"qperf:total","data":{"stream_mbps_received":2890.341,"stream_bytes_received":3614041924,"period":10003.087}}
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