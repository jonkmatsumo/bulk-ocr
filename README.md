# bulk-ocr

A Docker-only document processing pipeline that converts a directory of images into a deduplicated Markdown document using OCR.

## Quickstart

1. Place your images (JPEG/PNG) in the `./input` directory
2. Run: `docker compose up --build`
3. Outputs will appear in `./output`

## Expected Behavior (Milestone 0)

Currently, the pipeline only performs image discovery. It will:
- Scan the input directory for images (`.jpg`, `.jpeg`, `.png`)
- Log the number of images found
- Create the output directory if it doesn't exist

No OCR processing is performed yet. This is a scaffolding milestone.

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
  bulk-ocr run \
  --input /work/input \
  --out /work/output
```

### Local Development

```bash
# Build
go build ./cmd/pipeline

# Run
./pipeline run --input input --out output

# Subcommands
./pipeline version
./pipeline doctor
```

## Flags

- `--input` (default: `input`): Input directory containing images
- `--out` (default: `output`): Output directory for results
- `--keep-artifacts` (default: `true`): Keep intermediate artifacts
- `--lang` (default: `eng`): OCR language (not used in Milestone 0)

## Troubleshooting

### No images found

- Ensure images are in the `./input` directory (or the directory specified by `--input`)
- Supported extensions: `.jpg`, `.jpeg`, `.png` (case-insensitive)
- The pipeline searches recursively through subdirectories

### Docker build fails

- Ensure Docker is installed and running
- Check that `go.mod` exists and is valid
- Verify the Dockerfile syntax

### Permission errors

- Ensure the output directory is writable
- On Linux/Mac, you may need to adjust directory permissions

## Development Setup

### Prerequisites

- Go 1.23 or later
- Docker and Docker Compose
- Make (optional, for convenience targets)

### Local Development

```bash
# Format code
make fmt-fix

# Run tests
make test

# Run linter
make lint

# Build binary
make build

# Build Docker image
make docker-build
```

### Running Tests

```bash
# All tests
go test ./...

# With race detector
go test ./... -race

# With coverage
make test-coverage
```

## Project Structure

```
.
├── cmd/pipeline/      # CLI entry point
├── internal/          # Internal packages
│   ├── runner/        # External command execution
│   ├── ingest/        # Image discovery and ordering
│   ├── text/          # Text cleanup and chunking
│   ├── dedupe/        # Deduplication logic
│   └── report/        # JSON report generation
├── input/             # User-provided input directory
├── output/            # Generated output directory
├── testdata/          # Test fixtures
└── scripts/           # Utility scripts
```

## License

See LICENSE file for details.
