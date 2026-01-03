# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod ./
# Copy go.sum if it exists (optional - will be created by go mod download if missing)
COPY go.sum* ./
RUN go mod download

# Copy source code
# Strategy: Copy everything first, then verify critical directories exist
COPY . .

# Verify source directories exist (helps debug CI build context issues)
# This will fail early with a clear error message if directories are missing
RUN ls -la /build && \
    test -d cmd/pipeline || (echo "ERROR: cmd/pipeline directory not found in build context" && ls -la /build && exit 1) && \
    test -d internal || (echo "ERROR: internal directory not found in build context" && ls -la /build && exit 1) && \
    echo "Source directories verified successfully"

# Build binary
RUN go build -o /pipeline ./cmd/pipeline

# Runtime stage
FROM alpine:latest

WORKDIR /work

# Install required tools
# Note: tesseract-ocr may include English by default, but we'll verify and download if needed
RUN apk add --no-cache \
    tesseract-ocr \
    curl \
    ghostscript \
    poppler-utils \
    python3 \
    py3-pip \
    && pip3 install --break-system-packages --no-cache-dir ocrmypdf img2pdf \
    && mkdir -p /usr/share/tessdata \
    && (tesseract --list-langs 2>&1 | grep -q "^eng$" || \
        curl -L https://github.com/tesseract-ocr/tessdata/raw/main/eng.traineddata \
             -o /usr/share/tessdata/eng.traineddata) \
    && (test -f /usr/share/tessdata/osd.traineddata || \
        curl -L https://github.com/tesseract-ocr/tessdata/raw/main/osd.traineddata \
             -o /usr/share/tessdata/osd.traineddata) \
    && apk del curl \
    && tesseract --list-langs | grep -q "^eng$" || (echo "Warning: English language data not found" && exit 1)

# Copy binary from builder
COPY --from=builder /pipeline /pipeline

# Set entrypoint
ENTRYPOINT ["/pipeline"]

