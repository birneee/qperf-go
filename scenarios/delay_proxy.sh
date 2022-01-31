#!/bin/bash
source ./common.sh

trap "pkill -P $$" SIGINT

# Build qperf
build_qperf

setup_environment

RECEIVE_WINDOW=$(expr $BDP \* 3)

# Start server
sudo ip netns exec ns-server $QPERF_BIN server --tls-cert ../server.crt --tls-key ../server.key $QLOG &
SERVER_PID=$!

# Start client side proxy
sudo ip netns exec ns-client-side-proxy $QPERF_BIN proxy --tls-cert ../proxy.crt --tls-key ../proxy.key --server-side-max-receive-window $RECEIVE_WINDOW --log-prefix "client_side_proxy" $QLOG &
PROXY_PID=$!

# Start client
sudo ip netns exec ns-client $QPERF_BIN client --addr $SERVER_IP --proxy $CLIENT_SIDE_PROXY_IP -t 40 --tls-cert ../server.crt --tls-proxy-cert ../proxy.crt $QLOG $XSE &
CLIENT_PID=$!

wait $CLIENT_PID
sudo pkill -P $SERVER_PID
wait $SERVER_PID
sudo pkill -P $PROXY_PID
wait $PROXY_PID

cleanup_environment