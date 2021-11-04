#!/bin/bash
source ./common.sh

# Build qperf
build_qperf

# Add namespaces
sudo ip netns add ns-server
sudo ip netns add ns-client

# Add and link interfaces
sudo ip netns exec ns-client ip link add eth-client type veth peer name eth-server netns ns-server

# Assign IP adresses
sudo ip netns exec ns-server ip addr add 10.0.0.1/24 dev eth-server
sudo ip netns exec ns-client ip addr add 10.0.0.101/24 dev eth-client

# Set interfaces up
sudo ip netns exec ns-server ip link set dev eth-server up
sudo ip netns exec ns-client ip link set dev eth-client up

# Start server
sudo ip netns exec ns-server $QPERF_BIN -s --migrate 3 &
SERVER_PID=$!

# Start client
sudo ip netns exec ns-client $QPERF_BIN -c 10.0.0.1 &
CLIENT_PID=$!

wait $CLIENT_PID
sudo pkill -P $SERVER_PID
wait $SERVER_PID

# Delete namespaces
sudo ip netns delete ns-client
sudo ip netns delete ns-server