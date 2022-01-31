#!/bin/bash
source ./common.sh

trap "pkill -P $$" SIGINT

# Build qperf
build_qperf

setup_environment

# Start server
sudo ip netns exec ns-server $QPERF_BIN server --tls-cert ../server.crt --tls-key ../server.key $QLOG &
SERVER_PID=$!

# Start client
sudo ip netns exec ns-client $QPERF_BIN client --addr $SERVER_IP -t 40 --tls-cert ../server.crt $QLOG $XSE &
CLIENT_PID=$!

wait $CLIENT_PID
sudo pkill -P $SERVER_PID
wait $SERVER_PID

cleanup_environment