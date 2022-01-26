#!/bin/bash
source ./common.sh

trap "pkill -P $$" SIGINT

# Build qperf
build_qperf

setup_environment

CONGESTION_WINDOW=$(expr $MAX_IN_FLIGHT \* 95 / 100)
RECEIVE_WINDOW=$(expr $BDP \* 3)

# Start server
sudo ip netns exec ns-server $QPERF_BIN server --tls-cert ../server.crt --tls-key ../server.key $QLOG &
SERVER_PID=$!

# Start server-side proxy
sudo ip netns exec ns-server-side-proxy $QPERF_BIN proxy --tls-cert ../proxy.crt --tls-key ../proxy.key --client-side-min-congestion-window $CONGESTION_WINDOW --client-side-max-congestion-window $CONGESTION_WINDOW --log-prefix "server_side_proxy" $QLOG &
SERVER_SIDE_PROXY_PID=$!

# Start client-side proxy
sudo ip netns exec ns-client-side-proxy $QPERF_BIN proxy --tls-cert ../proxy.crt --tls-key ../proxy.key --next-proxy $SERVER_SIDE_PROXY_IP --0rtt --next-proxy-cert ../proxy.crt --server-side-initial-receive-window $RECEIVE_WINDOW --log-prefix "client_side_proxy" $QLOG &
CLIENT_SIDE_PROXY_PID=$!

# give server and proxies some time to setup e.g. to share 0-rtt information
sleep 2

# Start client
sudo ip netns exec ns-client $QPERF_BIN client --addr $SERVER_IP --proxy $CLIENT_SIDE_PROXY_IP -t 40 --tls-cert ../server.crt --tls-proxy-cert ../proxy.crt $QLOG &
CLIENT_PID=$!

wait $CLIENT_PID
sudo pkill -P $SERVER_PID
wait $SERVER_PID
sudo pkill -P $CLIENT_SIDE_PROXY_PID
wait $CLIENT_SIDE_PROXY_PID
sudo pkill -P $SERVER_SIDE_PROXY_PID
wait $SERVER_SIDE_PROXY_PID

cleanup_environment