FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive
ENV GO_VERSION=1.23.0
ENV GO_ARCH=amd64

RUN apt-get update && \
    apt-get install -y wget gnupg2 lsb-release supervisor git && \
    echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list && \
    wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - && \
    apt-get update && \
    apt-get install -y postgresql-17 && \
    apt-get clean

# Download and install Go
RUN wget https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz \
    && mkdir -p /usr/local/go \
    && tar -C /usr/local/go -xzf go${GO_VERSION}.linux-${GO_ARCH}.tar.gz \
    && rm -rf go${GO_VERSION}.linux-${GO_ARCH}.tar.gz

# Set environment variables for Go
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH=/go

RUN mkdir -p /var/log/supervisor

# PostgreSQL: listen on all interfaces & allow password auth
RUN sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" /etc/postgresql/17/main/postgresql.conf && \
    sed -i "s|#unix_socket_directories =.*|unix_socket_directories = '/tmp'|" /etc/postgresql/17/main/postgresql.conf && \
    echo "host all all all md5" >> /etc/postgresql/17/main/pg_hba.conf

RUN mkdir /meetnearme &&\
    cd /meetnearme &&\
    git clone https://github.com/meetnearme/api.git

WORKDIR /meetnearme/api

COPY seshujobs_init.sql /docker-entrypoint-initdb.d/seshujobs_init.sql

COPY init-postgres.sh /init-postgres.sh
RUN chmod +x /init-postgres.sh

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 5432

ENTRYPOINT ["/init-postgres.sh"]
