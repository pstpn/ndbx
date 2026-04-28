#!/bin/bash

set -e

CASSANDRA_HOST="${CASSANDRA_HOST:-cassandra}"
CASSANDRA_PORT="${CASSANDRA_PORT:-9042}"
TIMEOUT="${CASSANDRA_TIMEOUT:-60}"

start_time=$(date +%s)

echo "Waiting for Cassandra at $CASSANDRA_HOST:$CASSANDRA_PORT..."

while true; do
  if cqlsh "$CASSANDRA_HOST" "$CASSANDRA_PORT" -e "SELECT now() FROM system.local" > /dev/null 2>&1; then
    echo "Cassandra is healthy."
    exit 0
  fi

  current_time=$(date +%s)
  elapsed=$((current_time - start_time))

  if [ $elapsed -ge $TIMEOUT ]; then
    echo "Timeout waiting for Cassandra after ${TIMEOUT}s"
    exit 1
  fi

  echo "Cassandra is unavailable - retrying in 2s..."
  sleep 2
done
