FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y wget gnupg2 lsb-release supervisor && \
    echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list && \
    wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - && \
    apt-get update && \
    apt-get install -y postgresql-17 && \
    apt-get clean

RUN mkdir -p /var/log/supervisor

# PostgreSQL: listen on all interfaces & allow password auth
RUN sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" /etc/postgresql/17/main/postgresql.conf && \
    sed -i "s|#unix_socket_directories =.*|unix_socket_directories = '/tmp'|" /etc/postgresql/17/main/postgresql.conf && \
    echo "host all all all md5" >> /etc/postgresql/17/main/pg_hba.conf

COPY seshujobs_init.sql /docker-entrypoint-initdb.d/seshujobs_init.sql

COPY init-postgres.sh /init-postgres.sh
RUN chmod +x /init-postgres.sh

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 5432

ENTRYPOINT ["/init-postgres.sh"]
