# POSTGRES BUILD STEP
FROM postgres:17.4-alpine3.21 AS postgres-builder

RUN apk add --no-cache git build-base clang19 llvm19-dev

# WEAVIATE BUILD STAGE
FROM cr.weaviate.io/semitechnologies/weaviate:1.30.1 AS weaviate-builder

# Working with how reloading and building the go binary on host machine
FROM golang:1.23.8-alpine3.21


# Copy weaviate binary and dependencies from official iamge
COPY --from=weaviate-builder /bin/weaviate /usr/local/bin/weaviate
RUN mkdir -p /modules


RUN apk add --no-cache \
    git \
    build-base \
    supervisor \
    curl \
    ca-certificates \
    tzdata

# Create weaviate user and group
RUN addgroup -S weaviate && adduser -S weaviate -G weaviate

WORKDIR /go-app


# Make Static Dir for essential initialization and run time files
RUN mkdir /app-static
COPY go.mod go.sum .env /app-static/

RUN mkdir -p /run/supervisor /var/run/supervisor /var/log/supervisor && \
    # should think about these permissions for security
    chown root:root /run/supervisor /var/run/supervisor /var/log/supervisor && \
    chmod 755 /var/log/supervisor && \
    chmod 777 /var/run/supervisor && \
    chmod 777 /run/supervisor
    mkdir -p /var/lib/postgresql/data && \
    mkdir -p /var/lib/weaviate && \
    mkdir -p /var/log/supervisor && \
    mkdir -p /var/log/weaviate && \
    chown -R postgres:postgres /var/lib/postgresql/data && \
    chown -R weaviate:weaviate /var/lib/weaviate /modules /var/log/weaviate && \
    chown -R weaviate:weaviate /var/log/supervisor && \

VOLUME ["/var/lib/postgresql/data", "/var/lib/weaviate"]

COPY  ./docker_build/ ./

# Decide where to set these .env or here
ENV SST_STAGE=DEV
ENV DEPLOYMENT_TARGET=ACT

# Fake shims for lambda
ENV _LAMBDA_SERVER_PORT=""
ENV AWS_LAMBDA_RUNTIME_API=""

ENV POSTGRES_PASSWORD=postgres \
    POSTGRES_DB=postgres \
    POSTGRES_USER=postgres \
    WEAVIATE_HOST=0.0.0.0 \
    WEAVIATE_PORT=8080 \
    PGDATA=/var/lib/postgresql/data

# Set runtime environment variables (can be overridden by --env-file)
# Default Go app port should match the -p host:container mapping used
ENV APP_PORT=8000

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY weaviate.

EXPOSE 8000 5433 8080

# Start supervisord to manage all processes
CMD [ "/bin/sh", "-c", "cp /app-static/.env /go-app/ && exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf -n"]



