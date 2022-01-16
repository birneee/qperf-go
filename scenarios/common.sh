#!/bin/bash
QPERF_BIN="../qperf-go"
CLIENT_IP="10.0.0.101"
CLIENT_SIDE_PROXY_IP="10.0.0.102"
SERVER_IP="10.0.0.1"
SERVER_SIDE_PROXY_IP="10.0.0.2"
BANDWIDTH=100 # in mbit/s
DELAY=500 # in ms
RTT=`expr $DELAY \* 2` # in ms
PACKET_SIZE=1252 # in byte
LIMIT=`expr $BANDWIDTH \* $DELAY \* 1000 / $PACKET_SIZE / 8` # in packets


function build_qperf() {
  (cd .. ; go build qperf-go)
  exit_code=$?
  if [ $exit_code -ne 0 ]; then
    exit $exit_code
  fi
}

function setup_environment() {
  echo "Bandwidth: $BANDWIDTH Mbit/s"
  echo "RTT: $RTT ms"

  # Increase to recommended maximum buffer size (https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size)
  sudo sysctl -w net.core.rmem_max=2500000 >/dev/null
  sudo sysctl -w net.core.wmem_max=2500000 >/dev/null

  # Add namespaces
  sudo ip netns add ns-client
  sudo ip netns add ns-server
  sudo ip netns add ns-client-proxy
  sudo ip netns add ns-server-proxy
  sudo ip netns add ns-client-gateway
  sudo ip netns add ns-server-gateway

  # Add and link interfaces
  sudo ip netns exec ns-client ip link add eth-client type veth peer name eth-client netns ns-client-gateway
  sudo ip netns exec ns-server ip link add eth-server type veth peer name eth-server netns ns-server-gateway
  sudo ip netns exec ns-client-proxy ip link add eth-clientproxy type veth peer name eth-clientproxy netns ns-client-gateway
  sudo ip netns exec ns-server-proxy ip link add eth-serverproxy type veth peer name eth-serverproxy netns ns-server-gateway
  sudo ip netns exec ns-client-gateway ip link add eth-sat type veth peer name eth-sat netns ns-server-gateway

  # Add bridges
  sudo ip netns exec ns-client-gateway ip link add br-client type bridge
  sudo ip netns exec ns-server-gateway ip link add br-server type bridge

  # Connect links via bridges
  sudo ip netns exec ns-client-gateway ip link set eth-client master br-client
  sudo ip netns exec ns-client-gateway ip link set eth-clientproxy master br-client
  sudo ip netns exec ns-client-gateway ip link set eth-sat master br-client
  sudo ip netns exec ns-server-gateway ip link set eth-server master br-server
  sudo ip netns exec ns-server-gateway ip link set eth-serverproxy master br-server
  sudo ip netns exec ns-server-gateway ip link set eth-sat master br-server

  # Assign IP adresses
  sudo ip netns exec ns-server ip addr add $SERVER_IP/24 dev eth-server
  sudo ip netns exec ns-server-proxy ip addr add $SERVER_SIDE_PROXY_IP/24 dev eth-serverproxy
  sudo ip netns exec ns-client ip addr add $CLIENT_IP/24 dev eth-client
  sudo ip netns exec ns-client-proxy ip addr add $CLIENT_SIDE_PROXY_IP/24 dev eth-clientproxy

  # Set interfaces up
  sudo ip netns exec ns-server ip link set dev eth-server up
  sudo ip netns exec ns-client ip link set dev eth-client up
  sudo ip netns exec ns-client-proxy ip link set dev eth-clientproxy up
  sudo ip netns exec ns-server-proxy ip link set dev eth-serverproxy up
  sudo ip netns exec ns-client-gateway ip link set dev eth-client up
  sudo ip netns exec ns-client-gateway ip link set dev eth-sat up
  sudo ip netns exec ns-client-gateway ip link set dev eth-clientproxy up
  sudo ip netns exec ns-server-gateway ip link set dev eth-server up
  sudo ip netns exec ns-server-gateway ip link set dev eth-sat up
  sudo ip netns exec ns-server-gateway ip link set dev eth-serverproxy up

  # Set bridges up
  sudo ip netns exec ns-client-gateway ip link set br-client up
  sudo ip netns exec ns-server-gateway ip link set br-server up

  # Set Delay
  sudo ip netns exec ns-client-gateway tc qdisc replace dev eth-sat root netem limit $LIMIT delay ${DELAY}ms rate ${BANDWIDTH}mbit
  sudo ip netns exec ns-server-gateway tc qdisc replace dev eth-sat root netem limit $LIMIT delay ${DELAY}ms rate ${BANDWIDTH}mbit

  # Ping to resolve MAC through ARP
  sudo ip netns exec ns-client sudo ping $SERVER_IP -c 1 >/dev/null
  sudo ip netns exec ns-client sudo ping $CLIENT_SIDE_PROXY_IP -c 1 >/dev/null
  sudo ip netns exec ns-client-proxy sudo ping $SERVER_IP -c 1 >/dev/null
  sudo ip netns exec ns-client-proxy sudo ping $SERVER_SIDE_PROXY_IP -c 1 >/dev/null
  sudo ip netns exec ns-server-proxy sudo ping $SERVER_IP -c 1 >/dev/null
}

function cleanup_environment() {
  # Delete namespaces
  sudo ip netns delete ns-client
  sudo ip netns delete ns-server
  sudo ip netns delete ns-client-proxy
  sudo ip netns delete ns-server-proxy
  sudo ip netns delete ns-client-gateway
  sudo ip netns delete ns-server-gateway
}
