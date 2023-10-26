# Build Stage
FROM golang:1.20 AS app-build

WORKDIR /app

COPY go.mod go.mod
COPY go.sum .

RUN go mod download

COPY main.go .

RUN GOOS=linux go build -o main

# use ubuntu:focal if need more features in future
FROM ubuntu:focal

WORKDIR /app

# Install base Python deps
RUN apt-get update \
    && apt-get install -y python3.9 gcc make

RUN apt-get install -y python3-distutils python3-pip python3-apt sudo

RUN pip3 install awscli aws-sam-cli==1.99.0 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=app-build . .
COPY ./template.yaml ./app
COPY ./sam_cli_entrypoint.sh ./app
COPY ./event.json ./app

# run go app
CMD [ "main" ]

# error Error invoking remote method 'docker-start-container': Error: (HTTP code 400) unexpected - OCI runtime create failed: container_linux.go:380: starting container process caused: exec: "main": executable file not found in $PATH: unknown


# CMD [ "sam", "local", "start-api", "--host"]

