# Stage 1: Builder
FROM golang:1.26.3-bookworm AS builder

WORKDIR /build

RUN apt-get update && apt-get install -y ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/server ./cmd/api/

# Stage 2: Runtime
FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /build/server /app/server

RUN mkdir -p /app/certs && chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

CMD ["/app/server"]
