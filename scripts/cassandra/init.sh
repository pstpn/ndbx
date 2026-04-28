#!/bin/bash

set -e

CASSANDRA_HOST="${CASSANDRA_HOST:-cassandra}"
CASSANDRA_PORT="${CASSANDRA_PORT:-9042}"
CASSANDRA_KEYSPACE="${CASSANDRA_KEYSPACE:-ndbx}"
INIT_SCRIPT="/scripts/cassandra/init.cql"
TEMP_SCRIPT="/tmp/init.cql"

echo "Waiting for Cassandra to be ready..."
until cqlsh "$CASSANDRA_HOST" "$CASSANDRA_PORT" -e "DESCRIBE KEYSPACES" > /dev/null 2>&1; do
  echo "Cassandra is unavailable - sleeping"
  sleep 1
done

echo "Cassandra is ready. Preparing initialization script with keyspace=$CASSANDRA_KEYSPACE..."

sed "s/{{CASSANDRA_KEYSPACE}}/$CASSANDRA_KEYSPACE/g" "$INIT_SCRIPT" > "$TEMP_SCRIPT"

echo "Running initialization script..."

cqlsh "$CASSANDRA_HOST" "$CASSANDRA_PORT" -f "$TEMP_SCRIPT"

rm -f "$TEMP_SCRIPT"

echo "Cassandra initialization completed successfully."
