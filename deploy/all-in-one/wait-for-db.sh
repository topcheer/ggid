#!/bin/sh
set -e

# Wait for PostgreSQL to be ready, then exec the provided command
MAX_TRIES=60
TRIES=0

until pg_isready -h 127.0.0.1 -p 5432 -q; do
    TRIES=$((TRIES + 1))
    if [ "$TRIES" -ge "$MAX_TRIES" ]; then
        echo "Timed out waiting for PostgreSQL"
        exit 1
    fi
    echo "Waiting for PostgreSQL... ($TRIES/$MAX_TRIES)"
    sleep 1
done

echo "PostgreSQL is ready, starting: $*"
exec "$@"
