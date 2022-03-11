#!/bin/bash
source "${BASH_SOURCE%/*}/common.sh"

trap "pkill -P $$" SIGINT

setup_environment

MAX_RECEIVE_WINDOW=$(expr $BDP \* 3)

# Start server
sudo ip netns exec ns-server sudo -u "$USER" "$QPERF_BIN" server --tls-cert "$SERVER_CRT" --tls-key "$SERVER_KEY" --log-prefix "server" $QLOG &
SERVER_PID=$!

# Start client side proxy
sudo ip netns exec ns-client-side-proxy sudo -u "$USER" "$QPERF_BIN" proxy --tls-cert "$PROXY_CRT" --tls-key "$PROXY_KEY" --server-facing-max-receive-window "$MAX_RECEIVE_WINDOW" --log-prefix "client_side_proxy" --qlog-prefix "client_side_proxy" $QLOG &
PROXY_PID=$!

# Start client
sudo ip netns exec ns-client sudo -u "$USER" "$QPERF_BIN" client --addr "$SERVER_IP" --proxy "$CLIENT_SIDE_PROXY_IP" -t "$TIME" -i $INTERVAL --tls-cert "$SERVER_CRT" --tls-proxy-cert "$PROXY_CRT" --log-prefix "client" $QLOG $XSE $RAW &
CLIENT_PID=$!

wait $CLIENT_PID
pgrep -P $SERVER_PID | xargs -I {} pgrep -P {} | xargs -I {} kill {}
wait $SERVER_PID
pgrep -P $PROXY_PID | xargs -I {} pgrep -P {} | xargs -I {} kill {}
wait $PROXY_PID

cleanup_environment