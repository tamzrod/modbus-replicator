# Dockerfile
# Modbus Replicator – with baked-in example config

FROM golang:1.25.0-alpine AS build

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o replicator ./cmd/replicator


# ---- runtime image ----
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates

# binary
COPY --from=build /app/replicator /usr/local/bin/replicator

# default example config (baked in)
COPY --from=build /app/internal/config/example.yaml /config/replicator.yaml

# config directory exists by default
VOLUME ["/config"]

ENTRYPOINT ["/usr/local/bin/replicator"]
CMD ["/config/replicator.yaml"]
