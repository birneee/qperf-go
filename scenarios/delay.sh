#!/bin/bash
source ./common.sh

trap "pkill -P $$" SIGINT

# Build qperf
build_qperf

# Increase to recommended maximum buffer size (https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size)
sudo sysctl -w net.core.rmem_max=2500000 >/dev/null
sudo sysctl -w net.core.wmem_max=2500000 >/dev/null

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

# Set Delay
sudo ip netns exec ns-client tc qdisc replace dev eth-client root netem limit 4992 delay 500ms rate 100mbit
sudo ip netns exec ns-server tc qdisc replace dev eth-server root netem limit 4992 delay 500ms rate 100mbit

# Start server
sudo ip netns exec ns-server $QPERF_BIN server --tls-cert ../server.crt --tls-key ../server.key &
SERVER_PID=$!

# Start client
sudo ip netns exec ns-client sudo ping 10.0.0.1 -c 1 >/dev/null # because of ARP request/response
sudo ip netns exec ns-client $QPERF_BIN client --addr 10.0.0.1 -t 20 --tls-cert ../server.crt &
CLIENT_PID=$!

wait $CLIENT_PID
sudo pkill -P $SERVER_PID
wait $SERVER_PID

# Delete namespaces
sudo ip netns delete ns-client
sudo ip netns delete ns-server