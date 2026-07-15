#!/bin/bash
# Deploy website/ to Cloudflare Pages via direct upload API
# Uses wrangler for the actual upload

set -e

WEBSITE_DIR="/Users/zhanju/ggai/ggid/website"
PROJECT_NAME="ggid-dev"

echo "Deploying $WEBSITE_DIR to Cloudflare Pages project: $PROJECT_NAME"

# Use wrangler pages deploy with --commit-dirty
wrangler pages deploy "$WEBSITE_DIR" \
  --project-name="$PROJECT_NAME" \
  --branch=main \
  --commit-dirty \
  2>&1

echo "Deployment complete."
