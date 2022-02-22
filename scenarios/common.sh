#!/bin/bash
if [ -z $QPERF_BIN ]; then
  QPERF_BIN="${BASH_SOURCE%/*}/../qperf-go"
fi
if [ -z $CERT_DIR ]; then
  CERT_DIR="${BASH_SOURCE%/*}/.."
fi
SERVER_CRT="$CERT_DIR/server.crt"
SERVER_KEY="$CERT_DIR/server.key"
PROXY_CRT="$CERT_DIR/proxy.crt"
PROXY_KEY="$CERT_DIR/proxy.key"
CLIENT_IP="10.0.0.101"
CLIENT_SIDE_PROXY_IP="10.0.0.102"
SERVER_IP="10.0.0.1"
SERVER_SIDE_PROXY_IP="10.0.0.2"
if [ -z $BANDWIDTH ]; then
  BANDWIDTH=100 # in mbit/s
fi
if [ -z $RTT ]; then
  RTT=1000 # in ms
fi
if [ -z $INTERVAL ]; then
  INTERVAL=1 # in s
fi
BDP=$(expr $BANDWIDTH \* $RTT \* 1000 / 8) # in byte per second
MTU_SIZE=1280 # in byte
MAX_IN_FLIGHT=$(expr $BDP / $MTU_SIZE) # in packets, in both ways
PATH_BUFFER=$(expr $MAX_IN_FLIGHT / 2) # in packets
LIMIT=$(expr $MAX_IN_FLIGHT / 2 + $PATH_BUFFER) # in packets, one way
if [ "$QLOG" == "1" ]; then
  QLOG='--qlog'
else
  unset QLOG
fi
if [ "$XSE" == "1" ]; then
  XSE='--xse'
else
  unset XSE
fi
if [ "$RAW" == "1" ]; then
  RAW='--print-raw'
else
  unset RAW
fi

function setup_environment() {
  echo "Bandwidth: $BANDWIDTH Mbit/s"
  echo "RTT: $RTT ms"
  echo "BDP: $BDP B/s"
  echo "Max In-Flight Packets: $MAX_IN_FLIGHT"

  # Increase to recommended maximum buffer size (https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size)
  sudo sysctl -w net.core.rmem_max=2500000 >/dev/null
  sudo sysctl -w net.core.wmem_max=2500000 >/dev/null

  # Add namespaces
  sudo ip netns add ns-client
  sudo ip netns add ns-server
  sudo ip netns add ns-client-side-proxy
  sudo ip netns add ns-server-side-proxy
  sudo ip netns add ns-client-side-gateway
  sudo ip netns add ns-server-side-gateway

  # Add and link interfaces
  sudo ip netns exec ns-client ip link add eth-client type veth peer name eth-client netns ns-client-side-gateway
  sudo ip netns exec ns-server ip link add eth-server type veth peer name eth-server netns ns-server-side-gateway
  sudo ip netns exec ns-client-side-proxy ip link add eth-proxy type veth peer name eth-proxy netns ns-client-side-gateway
  sudo ip netns exec ns-server-side-proxy ip link add eth-proxy type veth peer name eth-proxy netns ns-server-side-gateway
  sudo ip netns exec ns-client-side-gateway ip link add eth-sat type veth peer name eth-sat netns ns-server-side-gateway

  # Add bridges
  sudo ip netns exec ns-client-side-gateway ip link add br-client type bridge
  sudo ip netns exec ns-server-side-gateway ip link add br-server type bridge

  # Connect links via bridges
  sudo ip netns exec ns-client-side-gateway ip link set eth-client master br-client
  sudo ip netns exec ns-client-side-gateway ip link set eth-proxy master br-client
  sudo ip netns exec ns-client-side-gateway ip link set eth-sat master br-client
  sudo ip netns exec ns-server-side-gateway ip link set eth-server master br-server
  sudo ip netns exec ns-server-side-gateway ip link set eth-proxy master br-server
  sudo ip netns exec ns-server-side-gateway ip link set eth-sat master br-server

  # Assign IP adresses
  sudo ip netns exec ns-server ip addr add $SERVER_IP/24 dev eth-server
  sudo ip netns exec ns-server-side-proxy ip addr add $SERVER_SIDE_PROXY_IP/24 dev eth-proxy
  sudo ip netns exec ns-client ip addr add $CLIENT_IP/24 dev eth-client
  sudo ip netns exec ns-client-side-proxy ip addr add $CLIENT_SIDE_PROXY_IP/24 dev eth-proxy

  # Set interfaces up
  sudo ip netns exec ns-server ip link set dev eth-server up
  sudo ip netns exec ns-client ip link set dev eth-client up
  sudo ip netns exec ns-client-side-proxy ip link set dev eth-proxy up
  sudo ip netns exec ns-server-side-proxy ip link set dev eth-proxy up
  sudo ip netns exec ns-client-side-gateway ip link set dev eth-client up
  sudo ip netns exec ns-client-side-gateway ip link set dev eth-sat up
  sudo ip netns exec ns-client-side-gateway ip link set dev eth-proxy up
  sudo ip netns exec ns-server-side-gateway ip link set dev eth-server up
  sudo ip netns exec ns-server-side-gateway ip link set dev eth-sat up
  sudo ip netns exec ns-server-side-gateway ip link set dev eth-proxy up

  # Set bridges up
  sudo ip netns exec ns-client-side-gateway ip link set br-client up
  sudo ip netns exec ns-server-side-gateway ip link set br-server up

  # Ping to resolve MAC through ARP
  sudo ip netns exec ns-client sudo ping $SERVER_IP -c 1 >/dev/null
  sudo ip netns exec ns-client sudo ping $CLIENT_SIDE_PROXY_IP -c 1 >/dev/null
  sudo ip netns exec ns-client-side-proxy sudo ping $SERVER_IP -c 1 >/dev/null
  sudo ip netns exec ns-client-side-proxy sudo ping $SERVER_SIDE_PROXY_IP -c 1 >/dev/null
  sudo ip netns exec ns-server-side-proxy sudo ping $SERVER_IP -c 1 >/dev/null

  # Set RTT
  sudo ip netns exec ns-client-side-gateway tc qdisc replace dev eth-sat root netem limit $LIMIT delay $(expr $RTT / 2)ms rate ${BANDWIDTH}mbit
  sudo ip netns exec ns-server-side-gateway tc qdisc replace dev eth-sat root netem limit $LIMIT delay $(expr $RTT / 2)ms rate ${BANDWIDTH}mbit

  # Set MTU
  sudo ip netns exec ns-client-side-gateway ip link set dev eth-sat mtu $MTU_SIZE
  sudo ip netns exec ns-server-side-gateway ip link set dev eth-sat mtu $MTU_SIZE
}

function cleanup_environment() {
  # Delete namespaces
  sudo ip netns delete ns-client
  sudo ip netns delete ns-server
  sudo ip netns delete ns-client-side-proxy
  sudo ip netns delete ns-server-side-proxy
  sudo ip netns delete ns-client-side-gateway
  sudo ip netns delete ns-server-side-gateway
}
