# Build stage with Go and Oracle Instant Client
FROM golang:1.23-bookworm AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    libaio1 \
    unzip \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Download Oracle Instant Client 23.6 (stable version with direct download URL)
RUN mkdir -p /opt/oracle && cd /opt/oracle && \
    curl -fsSL -o instantclient.zip "https://download.oracle.com/otn_software/linux/instantclient/2360000/instantclient-basiclite-linux.x64-23.6.0.24.10.zip" && \
    unzip -q instantclient.zip && \
    rm instantclient.zip && \
    mv instantclient_* instantclient && \
    echo /opt/oracle/instantclient > /etc/ld.so.conf.d/oracle-instantclient.conf && \
    ldconfig

ENV LD_LIBRARY_PATH=/opt/oracle/instantclient
ENV ORACLE_HOME=/opt/oracle/instantclient

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled (required for godror)
RUN CGO_ENABLED=1 GOOS=linux go build -o gprint ./cmd/server

# Runtime stage - use slim Debian for Oracle client compatibility
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    libaio1 \
    ca-certificates \
    tzdata \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Copy Oracle Instant Client from builder stage
COPY --from=builder /opt/oracle/instantclient /opt/oracle/instantclient
RUN echo /opt/oracle/instantclient > /etc/ld.so.conf.d/oracle-instantclient.conf && ldconfig

ENV LD_LIBRARY_PATH=/opt/oracle/instantclient
ENV TNS_ADMIN=/app/wallet

# Create non-root user
RUN useradd -r -s /bin/false appuser

# Copy binary from builder
COPY --from=builder /app/gprint .

# Create wallet and output directories
# Wallet contents should be provided via:
# - Environment variable ORACLE_WALLET_CONTENT (base64 encoded wallet zip), or
# - Volume mount at runtime: -v /path/to/wallet:/app/wallet
RUN mkdir -p /app/wallet /app/output && chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./gprint"]
