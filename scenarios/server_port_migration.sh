#!/bin/bash
source "${BASH_SOURCE%/*}/common.sh"

# Increase to recommended maximum buffer size (https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size)
sudo sysctl -w net.core.rmem_max=2500000 >/dev/null

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
sudo ip netns exec ns-server sudo -u "$USER" "$QPERF_BIN" server --migrate 3 --tls-cert "$SERVER_CRT" --tls-key "$SERVER_KEY" &
SERVER_PID=$!

# Start client
sudo ip netns exec ns-client sudo -u "$USER" "$QPERF_BIN" client --addr 10.0.0.1 --tls-cert "$SERVER_CRT" &
CLIENT_PID=$!

wait $CLIENT_PID
pgrep -P $SERVER_PID | xargs -I {} pgrep -P {} | xargs -I {} kill {}
wait $SERVER_PID

# Delete namespaces
sudo ip netns delete ns-client
sudo ip netns delete ns-server