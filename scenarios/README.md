# QPERF-GO Scenarios

## Delay Emulation

### Scripts

Following scripts require root privileges to create network namespaces.

- delay.sh
  - unmodified QUIC
- delay_optimized.sh
  - optimized congestion and receive window
- delay_proxy.sh
  - client-side proxy
  - optimized max receive window
- delay_two_proxies.sh
  - client-side and server-side proxy
  - optimized congestion and receive window

### Options

Following environment variables can be used to configure the scenarios.

- `RTT`
  - set the emulated RTT in ms
  - default: 1000 ms
- `BANDWIDTH`
  - set the emulated bandwidth in Mbit/s
  - default: 100 Mbit/s
- `QLOG`
  - enable qlog output for client, server and proxies (0 or 1)
  - default: 0
- `XSE`
  - enable XSE-QUIC extension (0 or 1)
  - default: 0

### Examples
```bash
RTT=250 BANDWIDTH=1000 QLOG=1 ./distributed_pep_static_cc.sh 
```
```bash
TIME=5 INTERVAL=0.5 ./no_pep.sh 
```

## Migration Emulation

TODO