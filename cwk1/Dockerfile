# Builder stage
FROM golang:1.24-bookworm AS builder

# Install build deps for CGO and sqlite3
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libsqlite3-dev \
    ca-certificates \
    pkg-config \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy sources
COPY . .

# Build with CGO enabled for sqlite3
ARG TARGETOS=linux
ARG TARGETARCH
ENV CGO_ENABLED=1
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}
RUN go build -ldflags "-s -w" -o /app/retaildb-service ./

# Final runtime image
FROM debian:bookworm-slim

# Install runtime dependencies (sqlite)
RUN apt-get update && apt-get install -y --no-install-recommends \
    libsqlite3-0 \
    ca-certificates \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary and assets
COPY --from=builder /app/retaildb-service /app/retaildb-service
COPY openapi.yaml /app/openapi.yaml

# Non-root user
RUN useradd --create-home --shell /bin/false appuser && chown appuser:appuser /app/retaildb-service /app/openapi.yaml
USER appuser

EXPOSE 8080

ENV PORT=8080
ENV RETAILDB_PATH=/app/retailDB.sqlite

ENTRYPOINT ["/app/retaildb-service"]
