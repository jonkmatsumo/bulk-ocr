# bulk-ocr

A Docker-only document processing pipeline that converts a directory of images into a deduplicated Markdown document using OCR.

## Overview

**Goal**: Process bulk image collections (screenshots, scanned documents, photos) through OCR and produce a single, deduplicated Markdown file with all extracted text.

**Current State**: The pipeline currently handles image discovery, deterministic ordering, and staging. OCR processing and text extraction are in development.

## Quickstart

```bash
# Place images in ./input directory
docker compose up --build

# Outputs appear in ./output
```

## How It Works

The pipeline processes images through these stages:

1. **Image Discovery**: Recursively scans for images (`.jpg`, `.jpeg`, `.png`)
2. **Deterministic Ordering**: Sorts images naturally (e.g., `IMG_9.jpg` before `IMG_10.jpg`)
3. **Staging**: Copies images to `preprocessed/` with sequential names
4. **OCR & Text Extraction**: *(In development)* Converts images to PDF, runs OCR, extracts text
5. **Deduplication**: *(In development)* Removes near-duplicate content using SimHash
6. **Markdown Output**: *(In development)* Produces final `.md` file with deduplicated text

## Usage

### Docker Compose (Recommended)

```bash
docker compose up --build
```

### Docker Run

```bash
docker build -t bulk-ocr .
docker run --rm \
  -v ./input:/work/input:ro \
  -v ./output:/work/output \
  bulk-ocr run --input /work/input --out /work/output
```

### Command Line Options

- `--input` (default: `input`): Input directory containing images
- `--out` (default: `output`): Output directory for results
- `--recursive` (default: `true`): Search subdirectories recursively
- `--keep-artifacts` (default: `true`): Keep intermediate processing files
- `--lang` (default: `eng`): OCR language code

### Subcommands

- `pipeline version`: Show version information
- `pipeline doctor`: Check toolchain health (verifies OCR tools are installed)

## Troubleshooting

**No images found**: Ensure images are in the input directory with supported extensions (`.jpg`, `.jpeg`, `.png`, case-insensitive).

**Docker build fails**: Verify Docker is running and `go.mod` is valid.

**Permission errors**: Ensure the output directory is writable.

## Development

Built with Go 1.23+. Uses Docker for all external dependencies (Tesseract, ocrmypdf, pdftotext, img2pdf).

```bash
# Run tests
make test

# Build locally
make build

# Check toolchain
make doctor
```
