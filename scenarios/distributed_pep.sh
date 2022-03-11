#!/bin/bash
source "${BASH_SOURCE%/*}/common.sh"

trap "pkill -P $$" SIGINT

setup_environment

INITIAL_CONGESTION_WINDOW=$(expr $MAX_IN_FLIGHT \* 95 / 100)
INITIAL_RECEIVE_WINDOW=$(expr $BDP \* 1)
MAX_RECEIVE_WINDOW=$(expr $BDP \* 3)

# Start server
sudo ip netns exec ns-server sudo -u "$USER" "$QPERF_BIN" server --tls-cert "$SERVER_CRT" --tls-key "$SERVER_KEY" --log-prefix "server" $QLOG &
SERVER_PID=$!

# Start server-side proxy
sudo ip netns exec ns-server-side-proxy sudo -u "$USER" "$QPERF_BIN" proxy --tls-cert "$PROXY_CRT" --tls-key "$PROXY_KEY" --client-facing-initial-congestion-window "$INITIAL_CONGESTION_WINDOW" --client-facing-initial-receive-window "$INITIAL_RECEIVE_WINDOW" --log-prefix "server_side_proxy" --qlog-prefix "server_side_proxy" $QLOG &
SERVER_SIDE_PROXY_PID=$!

# Start client-side proxy
sudo ip netns exec ns-client-side-proxy sudo -u "$USER" "$QPERF_BIN" proxy --tls-cert "$PROXY_CRT" --tls-key "$PROXY_KEY" --next-proxy "$SERVER_SIDE_PROXY_IP" --0rtt --next-proxy-cert "$PROXY_CRT" --server-facing-initial-receive-window "$INITIAL_RECEIVE_WINDOW" --server-facing-max-receive-window "$MAX_RECEIVE_WINDOW" --log-prefix "client_side_proxy" --qlog-prefix "client_side_proxy" $QLOG &
CLIENT_SIDE_PROXY_PID=$!

# give server and proxies some time to setup e.g. to share 0-rtt information
sleep 1

# Start client
sudo ip netns exec ns-client sudo -u "$USER" "$QPERF_BIN" client --addr "$SERVER_IP" --proxy "$CLIENT_SIDE_PROXY_IP" -t "$TIME" -i $INTERVAL --tls-cert "$SERVER_CRT" --tls-proxy-cert "$PROXY_CRT" --log-prefix "client" $QLOG $XSE $RAW &
CLIENT_PID=$!

wait $CLIENT_PID
pgrep -P $SERVER_PID | xargs -I {} pgrep -P {} | xargs -I {} kill {}
wait $SERVER_PID
pgrep -P $CLIENT_SIDE_PROXY_PID | xargs -I {} pgrep -P {} | xargs -I {} kill {}
wait $CLIENT_SIDE_PROXY_PID
pgrep -P $SERVER_SIDE_PROXY_PID | xargs -I {} pgrep -P {} | xargs -I {} kill {}
wait $SERVER_SIDE_PROXY_PID

cleanup_environment