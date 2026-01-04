package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonkmatsumo/bulk-ocr/internal/dedupe"
	"github.com/jonkmatsumo/bulk-ocr/internal/ingest"
	"github.com/jonkmatsumo/bulk-ocr/internal/pipeline"
	"github.com/jonkmatsumo/bulk-ocr/internal/report"
	"github.com/jonkmatsumo/bulk-ocr/internal/text"
)

const version = "0.0.1"

func main() {
	// Check if first arg is a subcommand (not a flag)
	// If so, we need to handle flag parsing differently
	subcommand := "run"
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		subcommand = args[0]
		args = args[1:]
		// Reconstruct os.Args without subcommand for flag parsing
		// This allows flags to come after subcommand: "pipeline run --input X"
		os.Args = append([]string{os.Args[0]}, args...)
	}

	var (
		inputDir         = flag.String("input", "input", "Input directory containing images")
		outputDir        = flag.String("out", "output", "Output directory for results")
		keepArtifacts    = flag.Bool("keep-artifacts", true, "Keep intermediate artifacts")
		lang             = flag.String("lang", "eng", "OCR language")
		recursive        = flag.Bool("recursive", true, "Recursively search subdirectories for images")
		pdfTimeout       = flag.Duration("pdf-timeout", 5*time.Minute, "Timeout for PDF synthesis")
		ocrTimeout       = flag.Duration("ocr-timeout", 10*time.Minute, "Timeout for OCR processing")
		extractTimeout   = flag.Duration("extract-timeout", 2*time.Minute, "Timeout for text extraction")
		minChunkChars    = flag.Int("min-chunk-chars", 60, "Minimum chunk size in characters")
		maxBlankLines    = flag.Int("max-blank-lines", 2, "Maximum consecutive blank lines to split on")
		emitChunksJSONL  = flag.Bool("emit-chunks-jsonl", true, "Emit debug JSONL file with chunks")
		chromeRegexFlags = flag.String("chrome-regex", "", "Custom chrome filtering regex pattern (can be repeated)")
		simhashK         = flag.Int("simhash-k", 5, "Character k-gram size for SimHash")
		simhashThreshold = flag.Int("simhash-threshold", 6, "Hamming distance threshold for SimHash")
		window           = flag.Int("window", 250, "Sliding window size for deduplication")
		dedupeMethod     = flag.String("dedupe", "simhash", "Deduplication method: exact, simhash, or both")
		markdownTitle    = flag.String("markdown-title", "Extracted Notes", "Title for Markdown document")
		includeChunkIDs  = flag.Bool("include-chunk-ids", false, "Include chunk IDs as HTML comments in Markdown")
	)

	flag.Parse()

	// Get remaining args after flag parsing
	remainingArgs := flag.Args()

	switch subcommand {
	case "run":
		// Collect chrome regex patterns (for now, single flag; can be extended to repeatable)
		chromePatterns := text.DefaultChromePatterns()
		if *chromeRegexFlags != "" {
			chromePatterns = append(chromePatterns, *chromeRegexFlags)
		}
		if err := runCommand(*inputDir, *outputDir, *keepArtifacts, *lang, *recursive, *pdfTimeout, *ocrTimeout, *extractTimeout, *minChunkChars, *maxBlankLines, *emitChunksJSONL, chromePatterns, *simhashK, *simhashThreshold, *window, *dedupeMethod, *markdownTitle, *includeChunkIDs); err != nil {
			log.Fatalf("error: %v", err)
		}
	case "doctor":
		doctorArgs := remainingArgs
		if err := doctorCommand(doctorArgs); err != nil {
			log.Fatalf("doctor failed: %v", err)
		}
	case "version":
		fmt.Printf("pipeline version %s\n", version)
		os.Exit(0)
	default:
		fmt.Printf("unknown subcommand: %s\n", subcommand)
		fmt.Println("Available subcommands: run, doctor, version")
		os.Exit(1)
	}
}

func runCommand(inputDir, outputDir string, keepArtifacts bool, lang string, recursive bool, pdfTimeout, ocrTimeout, extractTimeout time.Duration, minChunkChars, maxBlankLines int, emitChunksJSONL bool, chromePatterns []string, simhashK, simhashThreshold, window int, dedupeMethod, markdownTitle string, includeChunkIDs bool) error {
	// Validate input directory
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", inputDir)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Resolve absolute paths for logging
	absInput, err := filepath.Abs(inputDir)
	if err != nil {
		absInput = inputDir
	}

	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		absOutput = outputDir
	}

	// Enumerate images
	images, err := ingest.ListImages(inputDir, recursive)
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	log.Printf("input directory: %s", absInput)
	log.Printf("output directory: %s", absOutput)
	log.Printf("images found: %d", len(images))
	log.Printf("recursive: %v", recursive)
	log.Printf("keep artifacts: %v", keepArtifacts)
	log.Printf("language: %s", lang)

	if len(images) == 0 {
		log.Println("warning: no images found in input directory")
		return nil
	}

	// Stage images to preprocessed directory
	staged, err := ingest.StageImages(images, outputDir)
	if err != nil {
		return fmt.Errorf("failed to stage images: %w", err)
	}

	log.Printf("staged %d images to preprocessed/", len(staged))

	// Pipeline stage 1: Build PDF from staged images
	preprocessedDir := filepath.Join(outputDir, "preprocessed")
	log.Printf("Building PDF from %d images...", len(staged))
	start := time.Now()
	pdfPath, err := pipeline.BuildPDF(preprocessedDir, outputDir, pdfTimeout)
	if err != nil {
		return fmt.Errorf("PDF synthesis failed: %w", err)
	}
	log.Printf("PDF built: %s (took %v)", pdfPath, time.Since(start))

	// Pipeline stage 2: Run OCR on PDF
	log.Printf("Running OCR (language: %s)...", lang)
	start = time.Now()
	ocrPath, err := pipeline.OCRPDF(pdfPath, outputDir, lang, ocrTimeout)
	if err != nil {
		return fmt.Errorf("OCR failed: %w", err)
	}
	log.Printf("OCR completed: %s (took %v)", ocrPath, time.Since(start))

	// Cleanup combined.pdf if not keeping artifacts
	if !keepArtifacts {
		if err := pipeline.CleanupArtifact(pdfPath); err != nil {
			log.Printf("warning: failed to cleanup combined.pdf: %v", err)
		} else {
			log.Printf("cleaned up combined.pdf")
		}
	}

	// Pipeline stage 3: Extract text from OCR PDF
	log.Printf("Extracting text from OCR PDF...")
	start = time.Now()
	textPath, err := pipeline.ExtractText(ocrPath, outputDir, extractTimeout)
	if err != nil {
		return fmt.Errorf("text extraction failed: %w", err)
	}
	log.Printf("Text extracted: %s (took %v)", textPath, time.Since(start))

	// Get file size for logging
	if info, err := os.Stat(textPath); err == nil {
		log.Printf("extracted text size: %d bytes", info.Size())
	}

	// Cleanup combined_ocr.pdf if not keeping artifacts
	if !keepArtifacts {
		if err := pipeline.CleanupArtifact(ocrPath); err != nil {
			log.Printf("warning: failed to cleanup combined_ocr.pdf: %v", err)
		} else {
			log.Printf("cleaned up combined_ocr.pdf")
		}
	}

	// Pipeline stage 4: Chunk extracted text
	log.Printf("Chunking extracted text...")
	start = time.Now()
	extractedText, err := os.ReadFile(textPath)
	if err != nil {
		return fmt.Errorf("failed to read extracted text: %w", err)
	}

	rawChunks := text.ChunkText(string(extractedText), minChunkChars)
	log.Printf("Found %d chunks (raw)", len(rawChunks))

	// Apply chrome filtering
	filteredChunks := text.FilterChrome(rawChunks, chromePatterns, 100) // 100 chars max for chrome filtering
	log.Printf("Filtered to %d chunks (chrome)", len(filteredChunks))

	// Write JSONL debug output if enabled
	if emitChunksJSONL {
		chunksJSONLPath := filepath.Join(outputDir, "chunks_raw.jsonl")
		if err := text.WriteChunksJSONL(filteredChunks, chunksJSONLPath); err != nil {
			return fmt.Errorf("failed to write chunks JSONL: %w", err)
		}
		log.Printf("Writing chunks to chunks_raw.jsonl")
	}

	log.Printf("Chunking completed: %d chunks ready for deduplication (took %v)", len(filteredChunks), time.Since(start))

	// Pipeline stage 5: Deduplicate chunks
	log.Printf("Deduplicating chunks...")
	start = time.Now()

	// Create deduplication config
	dedupeConfig := dedupe.Config{
		Method:           dedupeMethod,
		SimHashK:         simhashK,
		SimHashThreshold: simhashThreshold,
		Window:           window,
	}
	dedupeConfig.Validate()

	dedupeResult := dedupe.Dedupe(filteredChunks, dedupeConfig)
	log.Printf("Input: %d chunks", dedupeResult.Stats.InputCount)
	log.Printf("Kept: %d chunks", dedupeResult.Stats.KeptCount)
	log.Printf("Dropped: %d chunks (%d exact, %d near-duplicates)", dedupeResult.Stats.DroppedCount, dedupeResult.Stats.ExactDups, dedupeResult.Stats.NearDups)

	// Write deduplication report
	reportPath := filepath.Join(outputDir, "dedupe_report.json")
	if err := report.WriteReport(dedupeResult, len(images), dedupeConfig, reportPath); err != nil {
		log.Printf("warning: failed to write deduplication report: %v", err)
	} else {
		log.Printf("Deduplication report written: %s", reportPath)
	}

	log.Printf("Deduplication completed (took %v)", time.Since(start))

	// Pipeline stage 6: Generate Markdown output
	log.Printf("Generating Markdown output...")
	start = time.Now()

	// Render Markdown from kept chunks
	markdownContent := text.RenderMarkdown(markdownTitle, dedupeResult.KeptChunks, includeChunkIDs)

	// Write Markdown file
	markdownPath := filepath.Join(outputDir, "result.md")
	if err := text.WriteMarkdown(markdownContent, markdownPath); err != nil {
		return fmt.Errorf("failed to write Markdown file: %w", err)
	}

	log.Printf("Markdown written: %s (%d chunks, took %v)", markdownPath, len(dedupeResult.KeptChunks), time.Since(start))

	log.Printf("Pipeline completed successfully. Final output: %s", markdownPath)
	return nil
}
