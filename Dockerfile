
# Stage 1: Build the Go binary
FROM golang:1.27.4-alpine3.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -o /app/server ./cmd/api

# Stage 2: Run the application
FROM alpine:3.24

WORKDIR /app

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

COPY --from=builder /app/server /app/server

USER app

EXPOSE 8080

CMD ["/app/server"]
