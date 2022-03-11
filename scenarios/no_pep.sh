#!/bin/bash
source "${BASH_SOURCE%/*}/common.sh"

trap "pkill -P $$" SIGINT

setup_environment

# Start server
sudo ip netns exec ns-server sudo -u "$USER" "$QPERF_BIN" server --tls-cert "$SERVER_CRT" --tls-key "$SERVER_KEY" --log-prefix="server" $QLOG &
SERVER_PID=$!

# Start client
sudo ip netns exec ns-client sudo -u "$USER" "$QPERF_BIN" client --addr "$SERVER_IP" -t "$TIME" -i $INTERVAL --tls-cert "$SERVER_CRT" --log-prefix "client" $QLOG $XSE $RAW &
CLIENT_PID=$!

wait $CLIENT_PID
pgrep -P $SERVER_PID | xargs -I {} pgrep -P {} | xargs -I {} kill {}
wait $SERVER_PID

cleanup_environment