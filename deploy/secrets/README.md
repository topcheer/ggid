# GGID Docker Secrets — Production Secret Management
#
# This file defines Docker secrets for production deployments.
# Secrets are mounted as files at /run/secrets/<name> inside containers.
#
# Usage:
#   1. Create secret files: echo -n "my-password" > /tmp/postgres_password.txt
#   2. docker secret create ggid_postgres_password /tmp/postgres_password.txt
#   3. Reference in docker-compose: secrets: [postgres_password]
#
# Alternatively, use Docker Compose secrets with file-based approach:
#   echo "POSTGRES_PASSWORD=my-secure-password" > .env

version: "3.9"
