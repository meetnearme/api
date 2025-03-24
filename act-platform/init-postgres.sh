#!/bin/bash

POSTGRES_USER=${POSTGRES_USER:-myuser}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-mypassword}
POSTGRES_DB=${POSTGRES_DB:-mydata}

# Create init user
su - postgres -c "/usr/lib/postgresql/17/bin/pg_ctl -D /var/lib/postgresql/17/main -o \"-c config_file=/etc/postgresql/17/main/postgresql.conf\" -w start"
su - postgres -c "psql -tc \"SELECT 1 FROM pg_roles WHERE rolname = '$POSTGRES_USER'\" | grep -q 1 || psql -c \"CREATE USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD';\""
su - postgres -c "psql -tc \"SELECT 1 FROM pg_database WHERE datname = '$POSTGRES_DB'\" | grep -q 1 || psql -c \"CREATE DATABASE $POSTGRES_DB OWNER $POSTGRES_USER;\""

# Run init SQL (if DB exists)
su - postgres -c "psql -d $POSTGRES_DB -f /docker-entrypoint-initdb.d/seshujobs_init.sql"
su - postgres -c "/usr/lib/postgresql/17/bin/pg_ctl -D /var/lib/postgresql/17/main stop"

exec /usr/bin/supervisord -n
