# Start from the official Golang base image
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY main.go .

RUN CGO_ENABLED=0 go build -o /daemon main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /daemon /app/daemon

CMD ["/app/daemon"]
