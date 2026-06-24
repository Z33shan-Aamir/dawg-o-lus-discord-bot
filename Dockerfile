FROM golang:1.26-alpine3.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bot ./cmd/main

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/bot .
CMD ["./bot"]
