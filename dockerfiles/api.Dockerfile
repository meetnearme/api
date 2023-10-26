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

RUN apt-get install -y python3-distutils python3-pip python3-apt

RUN pip3 install awscli aws-sam-cli==1.99.0 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=app-build . . 

# CMD [ "make", "_run"]

