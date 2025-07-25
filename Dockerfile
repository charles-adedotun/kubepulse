# Build stage for Go backend
FROM golang:1.23-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the Go binary with version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
    -o kubepulse ./cmd/kubepulse

# Build stage for React frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy package files
COPY frontend/package*.json ./
RUN npm ci

# Copy frontend source
COPY frontend/ .

# Build frontend
RUN npm run build

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S kubepulse && \
    adduser -u 1000 -S kubepulse -G kubepulse

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /app/kubepulse /app/kubepulse

# Copy frontend build
COPY --from=frontend-builder /app/frontend/dist /app/frontend/dist

# Change ownership
RUN chown -R kubepulse:kubepulse /app

USER kubepulse

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/kubepulse", "health"]

# Run the binary
ENTRYPOINT ["/app/kubepulse"]
CMD ["serve", "--port", "8080"]