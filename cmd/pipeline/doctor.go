package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jonkmatsumo/bulk-ocr/internal/runner"
)

// runnerInterface for mocking in tests
type runnerInterface interface {
	LookPath(bin string) (string, error)
	Run(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error)
}

// doctorCommand runs the doctor subcommand to validate the toolchain.
func doctorCommand(args []string) error {
	return doctorCommandWithRunner(args, runner.New())
}

// doctorCommandWithRunner allows injecting a mock runner for testing
func doctorCommandWithRunner(args []string, r runnerInterface) error {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	smoke := fs.Bool("smoke", false, "Run smoke test to verify end-to-end functionality")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	ctx := context.Background()

	log.Println("Doctor report:")

	// Define required tools
	tools := []struct {
		name    string
		bin     string
		version []string
	}{
		{"python3", "python3", []string{"--version"}},
		{"ocrmypdf", "ocrmypdf", []string{"--version"}},
		{"tesseract", "tesseract", []string{"--version"}},
		{"pdftotext", "pdftotext", []string{"-v"}},
	}

	var hasErrors bool

	// Check presence and versions
	for _, tool := range tools {
		// Check presence
		path, err := r.LookPath(tool.bin)
		if err != nil {
			log.Printf("- %s: MISSING", tool.name)
			hasErrors = true
			continue
		}

		// Get version
		opts := runner.RunOpts{
			Timeout:         10 * time.Second,
			StderrMode:      runner.StreamAndCapture, // pdftotext prints to stderr
			StdoutMode:      runner.StreamAndCapture,
			MaxCaptureBytes: 1024,
		}

		result, err := r.Run(ctx, tool.bin, tool.version, opts)
		if err != nil {
			log.Printf("- %s: ERROR (%s)", tool.name, err)
			hasErrors = true
			continue
		}

		// Extract version from output (stdout or stderr)
		version := extractVersion(result.Stdout + result.Stderr)
		if version == "" {
			version = "OK"
		}

		log.Printf("- %s: OK (%s) [%s]", tool.name, version, path)
	}

	// Optional: Check ghostscript
	gsPath, err := r.LookPath("gs")
	if err == nil {
		opts := runner.RunOpts{
			Timeout:    10 * time.Second,
			StdoutMode: runner.Capture,
			StderrMode: runner.Capture,
		}
		result, err := r.Run(ctx, "gs", []string{"--version"}, opts)
		if err == nil {
			version := extractVersion(result.Stdout + result.Stderr)
			if version == "" {
				version = "OK"
			}
			log.Printf("- ghostscript: OK (%s) [%s]", version, gsPath)
		}
	}

	// Smoke test
	if *smoke {
		log.Println("Running smoke test...")
		// Type assertion to *runner.Runner for runSmokeTest
		if realRunner, ok := r.(*runner.Runner); ok {
			if err := runSmokeTest(ctx, realRunner); err != nil {
				log.Printf("Smoke test: FAILED (%v)", err)
				// In test mode, return error instead of exiting
				if _, ok := r.(*runner.Runner); !ok {
					return fmt.Errorf("smoke test failed: %w", err)
				}
				os.Exit(2)
			}
			log.Println("Smoke test: PASSED")
		} else {
			// In tests, skip smoke test if runner is mocked
			log.Println("Smoke test: SKIPPED (mocked runner)")
		}
	} else {
		log.Println("Smoke test: SKIPPED (use --smoke to run)")
	}

	if hasErrors {
		// In test mode, return error instead of exiting
		// Check if we're in a test by checking if runner is mocked
		if _, ok := r.(*runner.Runner); !ok {
			// Mocked runner means we're in a test - return error instead of exiting
			return fmt.Errorf("doctor found errors: missing or failed tools")
		}
		os.Exit(1)
	}

	return nil
}

// extractVersion extracts a version string from command output.
func extractVersion(output string) string {
	// Try to find version patterns
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for common version patterns
		if strings.Contains(line, "version") || strings.Contains(line, "Version") {
			// Extract version number
			parts := strings.Fields(line)
			for i, part := range parts {
				if strings.Contains(strings.ToLower(part), "version") && i+1 < len(parts) {
					return parts[i+1]
				}
			}
			// Fallback: return the line
			if len(line) < 100 {
				return line
			}
		}
	}

	// Fallback: return first non-empty line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && len(line) < 100 {
			return line
		}
	}

	return ""
}

// runSmokeTest performs an end-to-end smoke test.
func runSmokeTest(ctx context.Context, r *runner.Runner) error {
	tmpDir, err := os.MkdirTemp("", "doctor-smoke-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			// Log but don't fail the test - cleanup errors are non-fatal
			log.Printf("warning: failed to clean up temp directory %s: %v", tmpDir, err)
		}
	}()

	// Generate test image
	testImage := tmpDir + "/test.png"
	if err := generateTestImage(testImage); err != nil {
		return fmt.Errorf("failed to generate test image: %w", err)
	}

	// Create PDF using img2pdf
	testPDF := tmpDir + "/test.pdf"
	opts := runner.RunOpts{
		Timeout:    2 * time.Minute,
		StdoutMode: runner.Capture,
		StderrMode: runner.Capture,
		Dir:        tmpDir,
	}

	result, err := r.Run(ctx, "python3", []string{"-m", "img2pdf", testImage, "-o", testPDF}, opts)
	if err != nil {
		return fmt.Errorf("img2pdf failed: %w (stderr: %s)", err, result.Stderr)
	}

	// Run OCR on PDF
	ocrPDF := tmpDir + "/test_ocr.pdf"
	opts.Timeout = 2 * time.Minute
	result, err = r.Run(ctx, "ocrmypdf", []string{"--deskew", "--rotate-pages", testPDF, ocrPDF}, opts)
	if err != nil {
		return fmt.Errorf("ocrmypdf failed: %w (stderr: %s)", err, result.Stderr)
	}

	// Extract text
	opts.Timeout = 2 * time.Minute
	result, err = r.Run(ctx, "pdftotext", []string{"-layout", ocrPDF, "-"}, opts)
	if err != nil {
		return fmt.Errorf("pdftotext failed: %w (stderr: %s)", err, result.Stderr)
	}

	// Verify output contains text
	if len(strings.TrimSpace(result.Stdout)) == 0 {
		return fmt.Errorf("pdftotext produced no output")
	}

	return nil
}

// generateTestImage creates a small test PNG image with text.
func generateTestImage(path string) error {
	// Use a simple approach: create a minimal PNG using Go's image package
	// For simplicity, we'll use a very basic approach or call a simple command
	// Since we need text in the image, we can use ImageMagick if available, or
	// create a simple colored image and let OCR try to read it

	// For now, create a simple approach: use python to create a test image
	// This is acceptable since we're already requiring python3
	ctx := context.Background()
	r := runner.New()

	// Create a simple test image using Python PIL
	pythonScript := fmt.Sprintf(`
from PIL import Image, ImageDraw, ImageFont
img = Image.new('RGB', (200, 50), color='white')
draw = ImageDraw.Draw(img)
try:
    font = ImageFont.truetype('/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf', 20)
except:
    font = ImageFont.load_default()
draw.text((10, 10), 'TEST', fill='black', font=font)
img.save('%s')
`, path)

	opts := runner.RunOpts{
		Timeout:    10 * time.Second,
		StdoutMode: runner.Capture,
		StderrMode: runner.Capture,
	}

	result, err := r.Run(ctx, "python3", []string{"-c", pythonScript}, opts)
	if err != nil {
		// Fallback: create a very simple image without text
		// This is a minimal valid PNG (1x1 white pixel)
		// OCR won't find text, but the pipeline will at least run
		minimalPNG := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 dimensions
			0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, // bit depth, color type, etc.
			0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, 0x54, // IDAT chunk
			0x08, 0x99, 0x01, 0x01, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01, // image data
			0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82, // IEND chunk
		}
		return os.WriteFile(path, minimalPNG, 0644)
	}

	_ = result
	return nil
}
