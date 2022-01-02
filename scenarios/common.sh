#!/bin/bash
QPERF_BIN="../qperf-go"

function build_qperf() {
  (cd .. ; go build qperf-go)
  exit_code=$?
  if [ $exit_code -ne 0 ]; then
    exit $exit_code
  fi
}

function setup_environment() {
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
  sudo ip netns exec ns-server ip link set dev lo up # loopback
  sudo ip netns exec ns-client ip link set dev eth-client up
  sudo ip netns exec ns-client ip link set dev lo up # loopback

  # Set Delay
  sudo ip netns exec ns-client tc qdisc replace dev eth-client root netem limit 4992 delay 500ms rate 100mbit
  sudo ip netns exec ns-server tc qdisc replace dev eth-server root netem limit 4992 delay 500ms rate 100mbit

  # Ping to resolve MAC through ARP
  sudo ip netns exec ns-client sudo ping 10.0.0.1 -c 1 >/dev/null
  sudo ip netns exec ns-server sudo ping 10.0.0.101 -c 1 >/dev/null
}

function cleanup_environment() {
  # Delete namespaces
  sudo ip netns delete ns-client
  sudo ip netns delete ns-server
}
