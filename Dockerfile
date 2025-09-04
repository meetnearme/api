FROM golang:1.24.3-alpine3.22 AS base

RUN apk add --no-cache \
  curl \
  ca-certificates \
  tzdata # this is necessary for the timezone dependencies that are not automatically available in the image

WORKDIR /go-app

COPY go.mod go.sum ./
RUN go mod download

# TODO: find a way to globally pin templ binary version
RUN go install github.com/a-h/templ/cmd/templ@v0.2.793
RUN go install github.com/air-verse/air@latest

COPY functions/ ./functions/
# COPY cmd/ ./cmd/


RUN set -e && \
# Fetch Cloudflare locations data
  cf_locations=$(curl -s -f -X GET https://speed.cloudflare.com/locations) && \
  \
  if [ $? -ne 0 ] || [ -z "$cf_locations" ]; then \
    echo "Failed to fetch Cloudflare locations or received empty response" && \
    exit 1; \
  fi && \
  \
  # Escape special characters in JSON
  escaped_locations=$(echo "$cf_locations" | sed 's/`/\\`/g' | sed 's/$/\\n/' | tr -d '\n') && \
  \
  # Replace the placeholder in the template file
  sed "s|<replace me>|$escaped_locations|" functions/gateway/helpers/cloudflare_locations_template > functions/gateway/helpers/cloudflare_locations.go

# not sure we need this was probably debugging
# print a list of all files that end with *templ.go
# RUN find . -name "*templ.go"

# Add lsof for port checking in development
RUN apk add --no-cache lsof

#   DEVELOPMENT STAGE
#
#
FROM base AS development

COPY . .

# Copy migration files to development stage
COPY migrations ./migrations

RUN /go/bin/templ generate

ENTRYPOINT ["/go/bin/air"]

#   BUILDER STAGE
#
# A separate builder stage
FROM base AS builder

COPY . .

RUN /go/bin/templ generate
# -ldflags="-w -s" strips debug symbols, making the binary smaller.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ./main ./functions/gateway/main.go

# PRODUCTION STAGE
#
#
FROM alpine:latest AS production
RUN apk add --no-cache \
  ca-certificates \
  tzdata # this is necessary for the timezone dependencies that are not automatically available in the image

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /go-app

COPY --from=builder /go-app/main /go-app/main
COPY --from=builder /go-app/migrations /go-app/migrations

RUN chown -R appuser:appgroup /go-app

USER appuser

CMD ["/go-app/main"]
