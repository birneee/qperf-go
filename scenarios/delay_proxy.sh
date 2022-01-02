#!/bin/bash
source ./common.sh

trap "pkill -P $$" SIGINT

# Build qperf
build_qperf

setup_environment

# Start server
sudo ip netns exec ns-server $QPERF_BIN server --tls-cert ../server.crt --tls-key ../server.key &
SERVER_PID=$!

# Start client side proxy
sudo ip netns exec ns-client $QPERF_BIN proxy --tls-cert ../proxy.crt --tls-key ../proxy.key --server-side-initial-receive-window 50MB --log-prefix "client_side_proxy" &
PROXY_PID=$!

# Start client
sudo ip netns exec ns-client sudo ping 10.0.0.1 -c 1 >/dev/null # because of ARP request/response
sudo ip netns exec ns-client $QPERF_BIN client --addr 10.0.0.1 --proxy 10.0.0.101 -t 20 --tls-cert ../server.crt --tls-proxy-cert ../proxy.crt &
CLIENT_PID=$!

wait $CLIENT_PID
sudo pkill -P $SERVER_PID
wait $SERVER_PID
sudo pkill -P $PROXY_PID
wait $PROXY_PID

cleanup_environment