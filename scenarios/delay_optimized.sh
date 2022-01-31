#!/bin/bash
source ./common.sh

trap "pkill -P $$" SIGINT

# Build qperf
build_qperf

setup_environment

CONGESTION_WINDOW=$(expr $MAX_IN_FLIGHT \* 95 / 100)
RECEIVE_WINDOW=$(expr $BDP \* 3)

# Start server
sudo ip netns exec ns-server $QPERF_BIN server --tls-cert ../server.crt --tls-key ../server.key --min-congestion-window $CONGESTION_WINDOW --max-congestion-window $CONGESTION_WINDOW $QLOG &
SERVER_PID=$!

# Start client
sudo ip netns exec ns-client $QPERF_BIN client --addr $SERVER_IP -t 40 --tls-cert ../server.crt --initial-receive-window $RECEIVE_WINDOW $QLOG $XSE &
CLIENT_PID=$!

wait $CLIENT_PID
sudo pkill -P $SERVER_PID
wait $SERVER_PID

cleanup_environment