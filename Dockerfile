# Postgres builder image with pgvector installed
FROM postgres:17.4-alpine3.21 AS postgres-builder

RUN apk add --no-cache git build-base clang19 llvm19-dev

# Clone pgvector and build it
WORKDIR /tmp
RUN git clone --branch v0.8.0 https://github.com/pgvector/pgvector.git
WORKDIR /tmp/pgvector
RUN make
RUN make install

# Python builder image
FROM python:3.12-alpine3.21 AS python-builder

# Install system dependencies
RUN apk add --no-cache build-base

# Final image
FROM postgres:17.4-alpine3.21

# Copy pgvector files from the builder
COPY --from=postgres-builder /usr/local/lib/postgresql/bitcode/vector.index.bc /usr/local/lib/postgresql/bitcode/vector.index.bc
COPY --from=postgres-builder /usr/local/lib/postgresql/vector.so /usr/local/lib/postgresql/vector.so
COPY --from=postgres-builder /usr/local/share/postgresql/extension /usr/local/share/postgresql/extension

# Install dependencies for Go and Python
RUN apk add --no-cache \
    python3 \
    py3-pip \
    build-base \
    libffi-dev \
    libstdc++ \
    bash \
    supervisor

COPY --from=python-builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=python-builder /usr/local/bin /usr/local/bin

COPY ./internal/database/postgres/ /python-postgres-adapter/

# Run pgvector_adapter.py to create the vector extension and test table and queries
RUN python3 /python-postgres-adapter/pgvector_adapter.py

RUN python3 -m venv /python-postgres-adapter/venv
RUN /python-postgres-adapter/venv/bin/pip install --upgrade pip
RUN /python-postgres-adapter/venv/bin/pip install -r /python-postgres-adapter/requirements.txt

# Create directory for postgres data
RUN mkdir -p /var/lib/postgresql/data && \
    chown -R postgres:postgres /var/lib/postgresql/data

# Set environment variables
ENV POSTGRES_PASSWORD=your_secure_password
ENV POSTGRES_DB=app_database

# Set up supervisord to manage multiple processes
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Expose ports
EXPOSE 5432

# Start supervisord to manage all processes
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
