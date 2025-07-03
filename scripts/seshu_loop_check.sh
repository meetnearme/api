#!/bin/sh

# Unbuffered output to container/supervisor logs
exec >> /proc/1/fd/1 2>> /proc/1/fd/2

STATE_FILE="/tmp/seshu_last_exec_state"

while true; do
  TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  CURRENT_HOUR=$(date -u +"%Y-%m-%dT%H")

  echo "[$TIMESTAMP] Starting scheduled check..."

  # Only proceed if this container is the active leader
  if [ "$IS_ACT_LEADER" != "true" ]; then
    echo "[$TIMESTAMP] Not the leader (IS_ACT_LEADER=$IS_ACT_LEADER). Skipping."
    sleep 30
    continue
  fi

  LAST_EXEC=""
  LAST_HOUR=""

  if [ -f "$STATE_FILE" ]; then
    . "$STATE_FILE"
  fi

  echo "[$TIMESTAMP] LAST_EXEC=$LAST_EXEC, LAST_HOUR=$LAST_HOUR, CURRENT_HOUR=$CURRENT_HOUR"

  SHOULD_TRIGGER=false
  if [ "$LAST_HOUR" != "$CURRENT_HOUR" ]; then
    if [ -z "$LAST_EXEC" ]; then
      SHOULD_TRIGGER=true
    else
      LAST_EXEC_EPOCH=$(date -d "$LAST_EXEC" +%s 2>/dev/null || echo 0)
      NOW_EPOCH=$(date +%s)
      if [ "$LAST_EXEC_EPOCH" -lt $((NOW_EPOCH - 3600)) ]; then
        SHOULD_TRIGGER=true
      fi
    fi
  fi

  if [ "$SHOULD_TRIGGER" = "true" ]; then
    echo "[$TIMESTAMP] Leader confirmed. Executing scheduled task..."

    # Write the updated state
    echo "LAST_EXEC=\"$TIMESTAMP\"" > "$STATE_FILE"
    echo "LAST_HOUR=\"$CURRENT_HOUR\"" >> "$STATE_FILE"

    # Placeholder for API call or further logic (e.g. curl to Go API)
    # curl -X POST http://localhost:8000/api/seshu/trigger-hour -H "Content-Type: application/json" -d "{\"hour\":\"$CURRENT_HOUR\"}"
  else
    echo "[$TIMESTAMP] No execution needed. Either same hour or recently executed."
  fi

  echo "[$TIMESTAMP] Sleeping 30 seconds..."
  sleep 30
done
