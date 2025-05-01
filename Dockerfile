# Postgres builder image with pgvector installed
FROM postgres:17.4-alpine3.21 AS postgres-builder

RUN apk add --no-cache git build-base
# WORKDIR /tmp
# RUN git clone --branch v0.8.0 https://github.com/pgvector/pgvector.git
# WORKDIR /tmp/pgvector
# RUN make && make install

# Golang builder image
FROM golang:1.23.2-alpine AS golang-builder
RUN apk add --no-cache build-base

# Final image
FROM postgres:17.4-alpine3.21

# Copy pgvector build
# COPY --from=postgres-builder /usr/local/lib/postgresql/bitcode/vector.index.bc /usr/local/lib/postgresql/bitcode/vector.index.bc
# COPY --from=postgres-builder /usr/local/lib/postgresql/vector.so /usr/local/lib/postgresql/vector.so
# COPY --from=postgres-builder /usr/local/share/postgresql/extension /usr/local/share/postgresql/extension

# Install necessary dependencies
RUN apk add --no-cache \
    bash \
    curl \
    go \
    nodejs \
    npm \
    supervisor \
    aws-cli \
    git \
    nano

# Clone API repo
RUN mkdir -p /meetnearme && \
    git clone https://github.com/meetnearme/api.git /meetnearme/api

WORKDIR /meetnearme/api

# Install Go templ
RUN go install github.com/a-h/templ/cmd/templ@v0.2.793

# Install Node dependencies
RUN npm install

# Setup supervisor directory
RUN mkdir -p /var/log/supervisor

# Copy init script and make it executable
RUN chmod +x /act-platform/initMNM.sh

# Move seshujobs_init.sql to the expected Postgres init directory
RUN cp /act-platform/seshujobs_init.sql /docker-entrypoint-initdb.d/seshujobs_init.sql

# Move supervisord.conf to supervisor config directory
RUN cp supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Expose relevant ports
EXPOSE 3000 5432

# Run init script (it will call supervisord)
ENTRYPOINT ["/act-platform/initMNM.sh"]
