#!/bin/bash

POSTGRES_USER=${SESHUJOBS_POSTGRES_USER:-myuser}
POSTGRES_PASSWORD=${SESHUJOBS_POSTGRES_PASSWORD:-mypassword}
POSTGRES_DB=${SESHUJOBS_POSTGRES_DB:-mydata}

# Start PostgreSQL temporarily for DB init
echo "[INFO] Starting PostgreSQL temporarily..."
su - postgres -c "/usr/local/bin/docker-entrypoint.sh postgres &"
sleep 5

# Create user & db
echo "[INFO] Creating PostgreSQL user/database..."
su - postgres -c "psql -tc \"SELECT 1 FROM pg_roles WHERE rolname = '$POSTGRES_USER'\" | grep -q 1 || psql -c \"CREATE USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD';\""
su - postgres -c "psql -tc \"SELECT 1 FROM pg_database WHERE datname = '$POSTGRES_DB'\" | grep -q 1 || psql -c \"CREATE DATABASE $POSTGRES_DB OWNER $POSTGRES_USER;\""

# Run init SQL
su - postgres -c "psql -d $POSTGRES_DB -f /docker-entrypoint-initdb.d/seshujobs_init.sql"

# Stop temporary Postgres
echo "[INFO] Stopping temporary PostgreSQL..."
su - postgres -c "pg_ctl -D /var/lib/postgresql/data stop"

# Export dummy AWS credentials (override in production)
echo "[INFO] Setting dummy AWS credentials for development..."
export AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID:-dummykey}
export AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY:-dummysecret}
export AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION:-us-east-1}

# Copy Cloudflare template manually (if required)
if [ -f functions/gateway/helpers/cloudflare_locations_template ]; then
  cp  functions/gateway/helpers/cloudflare_locations.go
  echo '[INFO] ⚠️ You must still edit cloudflare_locations.go to add actual Cloudflare JSON!'
fi

# Start the SST server (for first deployment)
echo "[INFO] Running initial deployment..."
npm run dev:sst

# Keep the container alive afterward
tail -f /dev/null
