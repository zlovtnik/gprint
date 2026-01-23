# Build stage - use Oracle Linux for native Oracle client compatibility
FROM golang:1.23-bookworm AS builder

WORKDIR /app

# Install Oracle Instant Client for build (required for godror CGO compilation)
RUN apt-get update && apt-get install -y --no-install-recommends \
    libaio1 \
    wget \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Download and install Oracle Instant Client Basic + SDK
RUN mkdir -p /opt/oracle && \
    cd /opt/oracle && \
    wget -q https://download.oracle.com/otn_software/linux/instantclient/2370000/instantclient-basic-linux.x64-23.7.0.25.01.zip && \
    wget -q https://download.oracle.com/otn_software/linux/instantclient/2370000/instantclient-sdk-linux.x64-23.7.0.25.01.zip && \
    unzip -q instantclient-basic-linux.x64-23.7.0.25.01.zip && \
    unzip -q instantclient-sdk-linux.x64-23.7.0.25.01.zip && \
    rm -f *.zip && \
    echo /opt/oracle/instantclient_23_7 > /etc/ld.so.conf.d/oracle-instantclient.conf && \
    ldconfig

ENV LD_LIBRARY_PATH=/opt/oracle/instantclient_23_7
ENV PKG_CONFIG_PATH=/opt/oracle/instantclient_23_7

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

# Copy Oracle Instant Client from builder
COPY --from=builder /opt/oracle/instantclient_23_7 /opt/oracle/instantclient_23_7
RUN echo /opt/oracle/instantclient_23_7 > /etc/ld.so.conf.d/oracle-instantclient.conf && ldconfig

ENV LD_LIBRARY_PATH=/opt/oracle/instantclient_23_7
ENV TNS_ADMIN=/app/wallet

# Create non-root user
RUN useradd -r -s /bin/false appuser

# Copy binary from builder
COPY --from=builder /app/gprint .

# Copy wallet directory (will be mounted or baked in)
COPY wallet/ /app/wallet/

# Create output directory for print jobs
RUN mkdir -p /app/output && chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./gprint"]
