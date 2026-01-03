# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod ./
COPY go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN go build -o /pipeline ./cmd/pipeline

# Runtime stage
FROM alpine:latest

WORKDIR /work

# Copy binary from builder
COPY --from=builder /pipeline /pipeline

# Set entrypoint
ENTRYPOINT ["/pipeline"]

