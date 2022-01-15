#!/bin/bash
source ./common.sh

trap "pkill -P $$" SIGINT

# Build qperf
build_qperf

setup_environment

# Start server
sudo ip netns exec ns-server $QPERF_BIN server --tls-cert ../server.crt --tls-key ../server.key &
SERVER_PID=$!

# Start server-side proxy
sudo ip netns exec ns-server $QPERF_BIN proxy --tls-cert ../proxy.crt --tls-key ../proxy.key --client-side-min-congestion-window 8000 --client-side-max-congestion-window 8000 --client-side-initial-receive-window 50MB --log-prefix "server_side_proxy" &
SERVER_SIDE_PROXY_PID=$!

# Start client-side proxy
sudo ip netns exec ns-client $QPERF_BIN proxy --tls-cert ../proxy.crt --tls-key ../proxy.key --next-proxy 10.0.0.1 --0rtt --next-proxy-cert ../proxy.crt --server-side-initial-receive-window 50MB --log-prefix "client_side_proxy" &
CLIENT_SIDE_PROXY_PID=$!

# give server and proxies some time to setup e.g. to share 0-rtt information
sleep 2

# Start client
sudo ip netns exec ns-client $QPERF_BIN client --addr 10.0.0.1 --proxy 10.0.0.101 -t 20 --tls-cert ../server.crt --tls-proxy-cert ../proxy.crt &
CLIENT_PID=$!

wait $CLIENT_PID
sudo pkill -P $SERVER_PID
wait $SERVER_PID
sudo pkill -P $CLIENT_SIDE_PROXY_PID
wait $CLIENT_SIDE_PROXY_PID
sudo pkill -P $SERVER_SIDE_PROXY_PID
wait $SERVER_SIDE_PROXY_PID

cleanup_environment