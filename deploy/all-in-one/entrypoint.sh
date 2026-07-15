#!/bin/sh
set -e

# Ensure log directories exist
mkdir -p /var/log/supervisor /var/log/ggid

# Run the requested command (supervisord)
exec "$@"
