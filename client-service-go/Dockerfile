# Start from the official Golang base image
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download

# Copy the source code
COPY main.go .

# Build the Go application
# CGO_ENABLED=0 is important for creating a static binary (no external dependencies)
# -ldflags="-s -w" reduces binary size by stripping debug info
RUN CGO_ENABLED=0 go build -o /client main.go

# --- STAGE 2: Create a minimal production image ---
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /client /app/client

# Run the executable
CMD ["/app/client"]