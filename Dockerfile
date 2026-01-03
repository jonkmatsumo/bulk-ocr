# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod ./
# Copy go.sum if it exists (optional - will be created by go mod download if missing)
COPY go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN go build -o /pipeline ./cmd/pipeline

# Runtime stage
FROM alpine:latest

WORKDIR /work

# Install required tools
RUN apk add --no-cache \
    tesseract-ocr \
    tesseract-ocr-eng \
    ghostscript \
    poppler-utils \
    python3 \
    py3-pip \
    && pip3 install --no-cache-dir ocrmypdf img2pdf

# Copy binary from builder
COPY --from=builder /pipeline /pipeline

# Set entrypoint
ENTRYPOINT ["/pipeline"]

