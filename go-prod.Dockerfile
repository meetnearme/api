FROM golang:1.23.8-alpine3.21 AS base

RUN apk add --no-cache \
  supervisor \
  curl \
  ca-certificates \
  tzdata # this is necessary for the timezone dependencies that are not automatically available in the image

WORKDIR /go-app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# -ldflags="-w -s" strips debug symbols, making the binary smaller.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./functions/gateway/main.go

FROM alpine:latest

RUN apk add --no-cache \
  supervisor \
  curl \
  ca-certificates \
  tzdata # this is necessary for the timezone dependencies that are not automatically available in the image

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /go-app

COPY --from=base /go-app /app/go-app

USER appuser

CMD ["/app/go-app/main"]
