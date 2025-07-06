#!/bin/sh

STATE_FILE="/tmp/last_update.txt"

exec >> /proc/1/fd/1 2>> /proc/1/fd/2

# Initialize the state file with current time if missing
if [ ! -f "$STATE_FILE" ]; then
  INIT_NOW=$(date -u +%s)
  echo "$INIT_NOW" > "$STATE_FILE"
  echo "$STATE_FILE not found, initializing with $INIT_NOW"
fi

while true; do
  # Only proceed if this container is the active leader
  if [ "$IS_ACT_LEADER" != "true" ]; then
    echo "Not the leader (IS_ACT_LEADER=$IS_ACT_LEADER). Skipping."
    sleep 30
    continue
  fi

  LAST_UPDATE=$(cat "$STATE_FILE")

  RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
    -d "{\"time\": $LAST_UPDATE}" http://localhost:8000/api/gather-seshu-jobs)

  if echo "$RESPONSE" | grep -q "successful"; then
    echo "[INFO] Job triggered successfully."
  else
    echo "[INFO] Skipped."
  fi

  NEW_NOW=$(date -u +%s)
  echo "$NEW_NOW" > "$STATE_FILE"

  sleep 30
done
