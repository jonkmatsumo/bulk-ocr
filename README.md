# bulk-ocr

A Docker-only document processing pipeline that converts a directory of images into a deduplicated Markdown document using OCR.

## Overview

**Goal**: Process bulk image collections (screenshots, scanned documents, photos) through OCR and produce a single, deduplicated Markdown file with all extracted text.

**Current State**: The pipeline handles the complete workflow from image discovery through Markdown output. All core stages are implemented: image discovery, deterministic ordering, staging, PDF synthesis, OCR processing, text extraction, text chunking, deduplication, and Markdown generation.

## Quickstart

```bash
# 1. Place your images in ./input directory (supports .jpg, .jpeg, .png)
mkdir -p input output

# 2. Run the pipeline
docker compose up --build

# 3. Check the results
cat output/result.md          # Final deduplicated Markdown output
cat output/dedupe_report.json # Deduplication statistics
```

**Test the setup first:**
```bash
# Run integration test to verify everything works
make integration
```

**What you'll get:**
- `output/result.md` - Final Markdown document with all extracted text
- `output/dedupe_report.json` - Statistics about duplicates removed
- `output/preprocessed/` - Staged images (if `--keep-artifacts=true`)

## How It Works

The pipeline processes images through these stages:

1. **Image Discovery**: Recursively scans for images (`.jpg`, `.jpeg`, `.png`)
2. **Deterministic Ordering**: Sorts images naturally (e.g., `IMG_9.jpg` before `IMG_10.jpg`)
3. **Staging**: Copies images to `preprocessed/` with sequential names
4. **PDF Synthesis**: Combines staged images into a single PDF using img2pdf
5. **OCR Processing**: Runs OCR on the PDF using ocrmypdf with deskew and rotation
6. **Text Extraction**: Extracts text from the OCR'd PDF using pdftotext
7. **Text Chunking**: Splits extracted text into paragraphs, normalizes for hashing, and filters UI artifacts
8. **Deduplication**: Removes near-duplicate content using SimHash with exact hash pre-check
9. **Markdown Output**: Produces final `result.md` file with deduplicated text

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
- `--keep-artifacts` (default: `true`): Keep intermediate processing files (combined.pdf, combined_ocr.pdf)
- `--lang` (default: `eng`): OCR language code
- `--pdf-timeout` (default: `5m`): Timeout for PDF synthesis
- `--ocr-timeout` (default: `10m`): Timeout for OCR processing
- `--extract-timeout` (default: `2m`): Timeout for text extraction
- `--min-chunk-chars` (default: `60`): Minimum chunk size in characters
- `--max-blank-lines` (default: `2`): Maximum consecutive blank lines to split on
- `--emit-chunks-jsonl` (default: `true`): Emit debug JSONL file with chunks
- `--chrome-regex`: Custom chrome filtering regex pattern (can be repeated)
- `--simhash-k` (default: `5`): Character k-gram size for SimHash
- `--simhash-threshold` (default: `6`): Hamming distance threshold for SimHash
- `--window` (default: `250`): Sliding window size for deduplication
- `--dedupe` (default: `simhash`): Deduplication method: exact, simhash, or both
- `--markdown-title` (default: `Extracted Notes`): Title for Markdown document
- `--include-chunk-ids` (default: `false`): Include chunk IDs as HTML comments in Markdown

### Subcommands

- `pipeline version`: Show version information
- `pipeline doctor`: Check toolchain health (verifies OCR tools are installed)

## Tuning Guide

The pipeline provides several knobs to tune output quality and performance. Here are common scenarios and recommended adjustments:

### Too Many Small Fragments

If your output contains many small, incomplete chunks:

- **Increase `--min-chunk-chars`**: Raise from default `60` to `100` or `150` to filter out shorter fragments
- **Example**: `--min-chunk-chars=100`

### False Positive Duplicates

If legitimate content is being marked as duplicates:

- **Lower `--simhash-threshold`**: Reduce from default `6` to `4` or `5` to be less aggressive (range: 0-64)
- **Switch to `--dedupe=exact`**: Only remove exact duplicates, not near-duplicates
- **Example**: `--simhash-threshold=4 --dedupe=exact`

### Duplicates Persist

If duplicates are not being caught:

- **Increase `--window`**: Raise from default `250` to `500` or `1000` to compare against more previous chunks
- **Switch to `--dedupe=both`**: Use both exact and SimHash methods for maximum deduplication
- **Example**: `--window=500 --dedupe=both`

### Too Many UI Artifacts

If browser chrome, timestamps, or system UI elements appear in output:

- **Add custom `--chrome-regex` patterns**: Define patterns specific to your screenshots
- **Example**: `--chrome-regex="\\d{2}:\\d{2}" --chrome-regex="battery|wifi"`
- **Note**: Patterns are matched against normalized (lowercase, no punctuation) text

### Performance Issues

If processing large image sets times out:

- **Increase timeouts**: Adjust `--pdf-timeout`, `--ocr-timeout`, or `--extract-timeout` as needed
- **Example**: `--ocr-timeout=20m` for very large PDFs
- **Disable debug output**: Set `--emit-chunks-jsonl=false` to reduce I/O

### Choosing Deduplication Method

- **`exact`**: Fastest, only removes identical chunks. Use when duplicates are exact copies.
- **`simhash`** (default): Balanced, removes near-duplicates. Best for most use cases.
- **`both`**: Most aggressive, uses both methods. Use when maximum deduplication is needed.

## Troubleshooting

### No Images Found

**Symptom**: Pipeline reports "no images found in input directory"

**Solutions**:
- Ensure images are in the input directory with supported extensions (`.jpg`, `.jpeg`, `.png`, case-insensitive)
- Check that `--recursive=true` if images are in subdirectories
- Verify file permissions allow reading the input directory

### Docker Build Fails

**Symptom**: `docker build` command fails with errors

**Solutions**:
- Verify Docker is running: `docker ps`
- Check that `go.mod` is valid: `go mod verify`
- Ensure network connectivity for downloading base images
- Review Docker logs for specific error messages

### Permission Errors

**Symptom**: Pipeline fails with "permission denied" errors

**Solutions**:
- Ensure the output directory is writable: `chmod 755 output`
- Check Docker volume mount permissions
- Verify user has write access to mounted directories

### Images Appear Rotated

**Symptom**: Text in output appears rotated or upside down

**Solutions**:
- The pipeline uses `ocrmypdf` with automatic rotation detection
- If rotation persists, images may need manual correction before processing
- Check image EXIF orientation data: `exiftool image.jpg`

### Empty OCR Output

**Symptom**: `result.md` is empty or contains very little text

**Solutions**:
- Verify images contain readable text (not just images/diagrams)
- Check OCR language matches image content: `--lang=eng` for English
- Ensure image quality is sufficient (not too blurry or low resolution)
- Review `extracted.txt` (if `--keep-artifacts=true`) to see raw OCR output
- Try increasing `--ocr-timeout` if processing is timing out

### Too Many UI Artifacts

**Symptom**: Output contains browser chrome, timestamps, battery indicators, etc.

**Solutions**:
- Add custom chrome filtering patterns: `--chrome-regex="\\d{2}:\\d{2}"`
- Increase `--min-chunk-chars` to filter out short UI elements
- Review `chunks_raw.jsonl` (if `--emit-chunks-jsonl=true`) to identify patterns
- Chrome filtering only applies to chunks under 100 characters by default

### Performance Issues

**Symptom**: Pipeline times out or runs very slowly

**Solutions**:
- Increase timeout values: `--pdf-timeout=10m --ocr-timeout=20m`
- Process images in smaller batches
- Disable debug output: `--emit-chunks-jsonl=false`
- Use `--keep-artifacts=false` to reduce disk I/O
- Check Docker resource limits (CPU, memory)

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
