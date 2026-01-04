package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jonkmatsumo/bulk-ocr/internal/runner"
)

// runnerInterface allows mocking the runner for testing
type runnerInterface interface {
	Run(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error)
}

// BuildPDF combines staged images into a single PDF using img2pdf.
// Takes staged images from preprocessedDir and writes combined.pdf to outputDir.
// Returns the path to the created PDF file.
func BuildPDF(preprocessedDir, outputDir string, timeout time.Duration) (string, error) {
	return buildPDFWithRunner(runner.New(), preprocessedDir, outputDir, timeout)
}

// buildPDFWithRunner is the internal implementation that accepts a runner interface for testing
func buildPDFWithRunner(r runnerInterface, preprocessedDir, outputDir string, timeout time.Duration) (string, error) {
	// List all image files in preprocessed directory
	files, err := filepath.Glob(filepath.Join(preprocessedDir, "*"))
	if err != nil {
		return "", fmt.Errorf("failed to list preprocessed images: %w", err)
	}

	// Filter to only image files and sort for deterministic order
	var imageFiles []string
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			imageFiles = append(imageFiles, f)
		}
	}

	if len(imageFiles) == 0 {
		return "", fmt.Errorf("no image files found in preprocessed directory: %s", preprocessedDir)
	}

	// Sort for deterministic ordering
	sort.Strings(imageFiles)

	// Build command: python3 -m img2pdf <files...> -o combined.pdf
	outputPath := filepath.Join(outputDir, "combined.pdf")
	args := append(imageFiles, "-o", outputPath)

	ctx := context.Background()
	opts := runner.RunOpts{
		Timeout:    timeout,
		StdoutMode: runner.StreamAndCapture,
		StderrMode: runner.StreamAndCapture,
	}

	result, err := r.Run(ctx, "python3", append([]string{"-m", "img2pdf"}, args...), opts)
	if err != nil {
		return "", fmt.Errorf("img2pdf failed: %w (stderr: %s)", err, result.Stderr)
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("img2pdf completed but output file not found: %s", outputPath)
	}

	return outputPath, nil
}

// OCRPDF runs OCR on a PDF file using ocrmypdf.
// Takes a PDF path and writes the OCR'd PDF to outputDir as combined_ocr.pdf.
// Returns the path to the created OCR PDF file.
func OCRPDF(pdfPath, outputDir, lang string, timeout time.Duration) (string, error) {
	return ocrPDFWithRunner(runner.New(), pdfPath, outputDir, lang, timeout)
}

// ocrPDFWithRunner is the internal implementation that accepts a runner interface for testing
func ocrPDFWithRunner(r runnerInterface, pdfPath, outputDir, lang string, timeout time.Duration) (string, error) {
	outputPath := filepath.Join(outputDir, "combined_ocr.pdf")

	// Build command: ocrmypdf --deskew --rotate-pages -l <lang> input.pdf output.pdf
	args := []string{
		"--deskew",
		"--rotate-pages",
		"-l", lang,
		pdfPath,
		outputPath,
	}

	ctx := context.Background()
	opts := runner.RunOpts{
		Timeout:    timeout,
		StdoutMode: runner.StreamAndCapture,
		StderrMode: runner.StreamAndCapture,
	}

	result, err := r.Run(ctx, "ocrmypdf", args, opts)
	if err != nil {
		return "", fmt.Errorf("ocrmypdf failed: %w (stderr: %s)", err, result.Stderr)
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("ocrmypdf completed but output file not found: %s", outputPath)
	}

	return outputPath, nil
}

// ExtractText extracts text from an OCR'd PDF using pdftotext.
// Takes a PDF path and writes extracted text to outputDir as extracted.txt.
// Validates that the extracted text is not empty (minimum 20 characters).
// Returns the path to the created text file.
func ExtractText(pdfPath, outputDir string, timeout time.Duration) (string, error) {
	return extractTextWithRunner(runner.New(), pdfPath, outputDir, timeout)
}

// extractTextWithRunner is the internal implementation that accepts a runner interface for testing
func extractTextWithRunner(r runnerInterface, pdfPath, outputDir string, timeout time.Duration) (string, error) {
	outputPath := filepath.Join(outputDir, "extracted.txt")

	// Build command: pdftotext -layout input.pdf output.txt
	args := []string{
		"-layout",
		pdfPath,
		outputPath,
	}

	ctx := context.Background()
	opts := runner.RunOpts{
		Timeout:    timeout,
		StdoutMode: runner.StreamAndCapture,
		StderrMode: runner.StreamAndCapture,
	}

	result, err := r.Run(ctx, "pdftotext", args, opts)
	if err != nil {
		return "", fmt.Errorf("pdftotext failed: %w (stderr: %s)", err, result.Stderr)
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("pdftotext completed but output file not found: %s", outputPath)
	}

	// Validate extracted text is not empty (minimum 20 characters)
	content, err := os.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to read extracted text: %w", err)
	}

	text := strings.TrimSpace(string(content))
	if len(text) < 20 {
		return "", fmt.Errorf("extracted text is too short (%d chars, minimum 20): likely OCR failure or empty PDF", len(text))
	}

	return outputPath, nil
}

// CleanupArtifact removes an artifact file if it exists.
// Returns an error only if the file exists and deletion fails.
func CleanupArtifact(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to clean up
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove artifact %s: %w", path, err)
	}

	return nil
}
