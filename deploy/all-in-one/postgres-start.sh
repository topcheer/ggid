#!/bin/sh
set -e

DATA_DIR=/var/lib/postgresql/data

# Initialize PostgreSQL data directory if empty
if [ ! -d "$DATA_DIR/base" ]; then
    echo "Initializing PostgreSQL data directory..."
    su - postgres -c "initdb -D $DATA_DIR"
    su - postgres -c "echo 'host all all 127.0.0.1/32 trust' >> $DATA_DIR/pg_hba.conf"
    su - postgres -c "echo 'local all all trust' >> $DATA_DIR/pg_hba.conf"

    # Start PostgreSQL temporarily to create role and database
    su - postgres -c "pg_ctl -D $DATA_DIR start -l /var/log/postgresql/postgresql.log"
    sleep 3
    su - postgres -c "psql -c \"CREATE USER ggid WITH PASSWORD 'ggid' superuser;\"" || echo "role may already exist"
    su - postgres -c "psql -c \"CREATE DATABASE ggid OWNER ggid;\"" || echo "database may already exist"
    su - postgres -c "pg_ctl -D $DATA_DIR stop"
fi

# Start PostgreSQL in foreground
exec su - postgres -c "postgres -D $DATA_DIR"
