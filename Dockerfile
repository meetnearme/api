FROM golang:1.24.3-alpine3.21

RUN apk add --no-cache git build-base

WORKDIR /go-app

COPY go.mod go.sum .env ./
RUN go mod download

COPY ./functions/ ./functions/

RUN go mod tidy

# Decide where to set these .env or here
ENV SST_STAGE=DEV
ENV DEPLOYMENT_TARGET=ACT
# Fake shims for lambda
ENV _LAMBDA_SERVER_PORT=""
ENV AWS_LAMBDA_RUNTIME_API=""

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go-app/main ./functions/gateway/main.go

# EXPOSE 8000
ENTRYPOINT ["./main"]

