FROM golang:1.24-alpine AS builder

# Enable Go modules and static binary
ENV CGO_ENABLED=0 \
    GO111MODULE=on

WORKDIR /app

# Copy go.mod and go.sum separately to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary with optimizations
RUN go build -ldflags="-s -w" -o /writer main.go


FROM alpine:3.20

WORKDIR /app


# Copy only the compiled binary
COPY --from=builder /writer .

# Set default environment variables (can be overridden at runtime)
ENV INFLUXDB_HOST=http://influxdb:8086 \
    NATS_URL=nats://nats:4222

# Execute the binary
ENTRYPOINT ["/app/writer"]