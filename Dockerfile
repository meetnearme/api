# Working with how reloading and building the go binary on host machine
FROM golang:1.23.8-alpine3.21

RUN apk add --no-cache \
    git \
    build-base \
    supervisor \
    curl \
    ca-certificates \
    tzdata

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

COPY  ./docker_build/ ./

# Decide where to set these .env or here
ENV SST_STAGE=DEV
ENV DEPLOYMENT_TARGET=ACT

# Fake shims for lambda
ENV _LAMBDA_SERVER_PORT=""
ENV AWS_LAMBDA_RUNTIME_API=""

# Set runtime environment variables (can be overridden by --env-file)
# Default Go app port should match the -p host:container mapping used
ENV APP_PORT=8000

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 8000 5433 8080

# Start supervisord to manage all processes
CMD [ "/bin/sh", "-c", "cp /app-static/.env /go-app/ && exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf -n"]



