#!/bin/bash

set -e

CASSANDRA_HOST="${CASSANDRA_HOST:-cassandra}"
CASSANDRA_PORT="${CASSANDRA_PORT:-9042}"
INIT_SCRIPT="/scripts/cassandra/init.cql"

echo "Waiting for Cassandra to be ready..."
until cqlsh "$CASSANDRA_HOST" "$CASSANDRA_PORT" -e "DESCRIBE KEYSPACES" > /dev/null 2>&1; do
  echo "Cassandra is unavailable - sleeping"
  sleep 1
done

echo "Cassandra is ready. Running initialization script..."

cqlsh "$CASSANDRA_HOST" "$CASSANDRA_PORT" -f "$INIT_SCRIPT"

echo "Cassandra initialization completed successfully."
