# syntax=docker/dockerfile:1

# ---------- Stage 1: build ----------
FROM golang:1.26-alpine AS builder
WORKDIR /src

# Cache dependencies first — only re-runs when go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO disabled => fully static binary that runs on a bare Alpine image.
# -trimpath and -s -w strip build paths and debug symbols.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/auctasy-server ./cmd/server

# ---------- Stage 2: runtime ----------
FROM alpine:3.20
RUN adduser -D -u 10001 appuser && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/auctasy-server ./auctasy-server

USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["./auctasy-server"]
