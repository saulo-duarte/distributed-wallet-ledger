# syntax=docker/dockerfile:1

# Stage 1: Build binary statically
FROM golang:alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /app/bin/api \
    ./cmd/api

# Stage 2: Minimal non-root runtime image
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/migrations /app/migrations

USER nonroot:nonroot

EXPOSE 8082

ENTRYPOINT ["/app/api"]
