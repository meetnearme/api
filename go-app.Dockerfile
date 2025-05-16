FROM golang:1.23.8-alpine3.21 AS base

RUN apk add --no-cache \
  curl \
  ca-certificates \
  tzdata # this is necessary for the timezone dependencies that are not automatically available in the image

WORKDIR /go-app

RUN mkdir -p /go-app/docker_build

# The go binary will be mounted from ./docker_build to continue enabling the watchGolang script

CMD ["/go-app/docker_build/main"]


