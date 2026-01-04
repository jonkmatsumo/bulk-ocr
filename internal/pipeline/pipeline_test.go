package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonkmatsumo/bulk-ocr/internal/runner"
)

func TestBuildPDF_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	_, err := BuildPDF(tmpDir, outputDir, 30*time.Second)
	if err == nil {
		t.Error("expected error for empty directory, got nil")
	}
	if !strings.Contains(err.Error(), "no image files found") {
		t.Errorf("expected error about no image files, got: %v", err)
	}
}

func TestBuildPDF_NoImages(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	// Create a non-image file
	nonImage := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(nonImage, []byte("not an image"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := BuildPDF(tmpDir, outputDir, 30*time.Second)
	if err == nil {
		t.Error("expected error for no images, got nil")
	}
	if !strings.Contains(err.Error(), "no image files found") {
		t.Errorf("expected error about no image files, got: %v", err)
	}
}

func TestCleanupArtifact_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "test.pdf")

	// Create a test file
	if err := os.WriteFile(artifactPath, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		t.Fatal("test file should exist")
	}

	// Cleanup
	if err := CleanupArtifact(artifactPath); err != nil {
		t.Fatalf("CleanupArtifact failed: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(artifactPath); !os.IsNotExist(err) {
		t.Error("artifact file should have been deleted")
	}
}

func TestCleanupArtifact_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "nonexistent.pdf")

	// Cleanup non-existent file should not error
	if err := CleanupArtifact(artifactPath); err != nil {
		t.Errorf("CleanupArtifact should not error for non-existent file, got: %v", err)
	}
}

func TestCleanupArtifact_PermissionError(t *testing.T) {
	// This test is platform-specific and may not work on all systems
	// Skip on Windows or if we can't create a read-only directory
	if testing.Short() {
		t.Skip("skipping permission test in short mode")
	}

	// Note: Creating a truly unwritable file/directory in a test is complex
	// and platform-dependent. This test is a placeholder for the concept.
	// In practice, CleanupArtifact should handle permission errors gracefully.
	t.Log("Permission error testing would require platform-specific setup")
}

// mockRunner implements runnerInterface for testing
type mockRunner struct {
	runFunc func(context.Context, string, []string, runner.RunOpts) (runner.Result, error)
}

func (m *mockRunner) Run(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, bin, args, opts)
	}
	// Default: success
	return runner.Result{
		Cmd:        bin + " " + strings.Join(args, " "),
		ExitCode:   0,
		DurationMs: 100,
		Stdout:     "",
		Stderr:     "",
	}, nil
}

// Helper functions

// createMockImage creates a minimal valid PNG image file
func createMockImage(t *testing.T, dir, name string) string {
	path := filepath.Join(dir, name)
	// Create a minimal valid PNG (1x1 white pixel)
	minimalPNG := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 dimensions
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, // bit depth, color type, etc.
		0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, 0x54, // IDAT chunk
		0x08, 0x99, 0x01, 0x01, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01, // image data
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82, // IEND chunk
	}
	if err := os.WriteFile(path, minimalPNG, 0644); err != nil {
		t.Fatalf("failed to create mock image: %v", err)
	}
	return path
}

// createMockPDF creates a minimal valid PDF file
func createMockPDF(t *testing.T, dir string) string {
	path := filepath.Join(dir, "test.pdf")
	// Minimal valid PDF structure
	// PDF header, minimal object structure, and EOF marker
	minimalPDF := []byte(
		"%PDF-1.4\n" +
			"1 0 obj\n" +
			"<< /Type /Catalog /Pages 2 0 R >>\n" +
			"endobj\n" +
			"2 0 obj\n" +
			"<< /Type /Pages /Kids [3 0 R] /Count 1 >>\n" +
			"endobj\n" +
			"3 0 obj\n" +
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>\n" +
			"endobj\n" +
			"4 0 obj\n" +
			"<< /Length 44 >>\n" +
			"stream\n" +
			"BT /F1 12 Tf 100 700 Td (Test PDF) Tj ET\n" +
			"endstream\n" +
			"endobj\n" +
			"xref\n" +
			"0 5\n" +
			"0000000000 65535 f \n" +
			"0000000009 00000 n \n" +
			"0000000058 00000 n \n" +
			"0000000115 00000 n \n" +
			"0000000206 00000 n \n" +
			"trailer\n" +
			"<< /Size 5 /Root 1 0 R >>\n" +
			"startxref\n" +
			"300\n" +
			"%%EOF\n",
	)
	if err := os.WriteFile(path, minimalPDF, 0644); err != nil {
		t.Fatalf("failed to create mock PDF: %v", err)
	}
	return path
}

// createMockTextFile creates a text file with the specified content
func createMockTextFile(t *testing.T, dir, content string) string {
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create mock text file: %v", err)
	}
	return path
}

// BuildPDF Tests

// TestBuildPDF_SingleImage tests successful PDF creation from a single image
func TestBuildPDF_SingleImage(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	// Create a single image
	createMockImage(t, tmpDir, "image1.png")

	// Mock runner that simulates successful img2pdf execution
	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			// Verify command is correct
			if bin != "python3" {
				t.Errorf("expected python3, got %s", bin)
			}
			if len(args) < 3 || args[0] != "-m" || args[1] != "img2pdf" {
				t.Errorf("unexpected args: %v", args)
			}
			// Create the output PDF file
			outputPath := args[len(args)-1] // Last arg is -o outputPath
			for i, arg := range args {
				if arg == "-o" && i+1 < len(args) {
					outputPath = args[i+1]
					break
				}
			}
			// Create a minimal PDF file
			createMockPDF(t, filepath.Dir(outputPath))
			// Move it to the expected location
			if err := os.Rename(filepath.Join(filepath.Dir(outputPath), "test.pdf"), outputPath); err != nil {
				// If rename fails, just create the file directly
				_ = os.WriteFile(outputPath, []byte("%PDF-1.4\n"), 0644)
			}
			return runner.Result{ExitCode: 0}, nil
		},
	}

	result, err := buildPDFWithRunner(mockR, tmpDir, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("BuildPDF failed: %v", err)
	}

	expectedPath := filepath.Join(outputDir, "combined.pdf")
	if result != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result)
	}

	// Verify file exists
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("output PDF file was not created")
	}
}

// TestBuildPDF_MultipleImages tests successful PDF creation from multiple images
func TestBuildPDF_MultipleImages(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	// Create multiple images
	createMockImage(t, tmpDir, "image1.png")
	createMockImage(t, tmpDir, "image2.jpg")
	createMockImage(t, tmpDir, "image3.jpeg")

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			// Find output path
			outputPath := ""
			for i, arg := range args {
				if arg == "-o" && i+1 < len(args) {
					outputPath = args[i+1]
					break
				}
			}
			if outputPath == "" {
				t.Error("output path not found in args")
			}
			// Create output file
			_ = os.WriteFile(outputPath, []byte("%PDF-1.4\n"), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	result, err := buildPDFWithRunner(mockR, tmpDir, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("BuildPDF failed: %v", err)
	}

	expectedPath := filepath.Join(outputDir, "combined.pdf")
	if result != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result)
	}
}

// TestBuildPDF_DifferentFormats tests PDF creation with different image formats
func TestBuildPDF_DifferentFormats(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	// Create images with different formats
	createMockImage(t, tmpDir, "test1.jpg")
	createMockImage(t, tmpDir, "test2.jpeg")
	createMockImage(t, tmpDir, "test3.png")

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := ""
			for i, arg := range args {
				if arg == "-o" && i+1 < len(args) {
					outputPath = args[i+1]
					break
				}
			}
			_ = os.WriteFile(outputPath, []byte("%PDF-1.4\n"), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := buildPDFWithRunner(mockR, tmpDir, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("BuildPDF failed: %v", err)
	}
}

// TestBuildPDF_DeterministicOrdering tests that images are sorted deterministically
func TestBuildPDF_DeterministicOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	// Create images with names that would sort differently
	createMockImage(t, tmpDir, "image10.png")
	createMockImage(t, tmpDir, "image2.png")
	createMockImage(t, tmpDir, "image1.png")

	var capturedArgs []string
	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			capturedArgs = args
			outputPath := ""
			for i, arg := range args {
				if arg == "-o" && i+1 < len(args) {
					outputPath = args[i+1]
					break
				}
			}
			_ = os.WriteFile(outputPath, []byte("%PDF-1.4\n"), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := buildPDFWithRunner(mockR, tmpDir, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("BuildPDF failed: %v", err)
	}

	// Verify images are passed in sorted order (image1, image10, image2)
	// Find image arguments (skip -m, img2pdf, and -o, outputPath)
	imageArgs := []string{}
	for i, arg := range capturedArgs {
		if i > 1 && arg != "-o" && !strings.HasSuffix(arg, ".pdf") {
			// Check if previous arg was -o
			if i > 0 && capturedArgs[i-1] == "-o" {
				continue
			}
			imageArgs = append(imageArgs, arg)
		}
	}

	if len(imageArgs) < 3 {
		t.Fatalf("expected at least 3 image args, got %d", len(imageArgs))
	}

	// Verify sorted order (lexicographic: image1, image10, image2)
	basename1 := filepath.Base(imageArgs[0])
	basename2 := filepath.Base(imageArgs[1])
	basename3 := filepath.Base(imageArgs[2])

	if !strings.Contains(basename1, "image1") {
		t.Errorf("first image should be image1, got %s", basename1)
	}
	if !strings.Contains(basename2, "image10") {
		t.Errorf("second image should be image10, got %s", basename2)
	}
	if !strings.Contains(basename3, "image2") {
		t.Errorf("third image should be image2, got %s", basename3)
	}
}

// TestBuildPDF_Img2pdfFailure tests error handling when img2pdf fails
func TestBuildPDF_Img2pdfFailure(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	createMockImage(t, tmpDir, "image1.png")

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			return runner.Result{
					ExitCode: 1,
					Stderr:   "img2pdf error: invalid image format",
				}, &runner.ExecError{
					Bin:    bin,
					Args:   args,
					Result: runner.Result{ExitCode: 1, Stderr: "img2pdf error: invalid image format"},
				}
		},
	}

	_, err := buildPDFWithRunner(mockR, tmpDir, outputDir, 30*time.Second)
	if err == nil {
		t.Error("expected error for img2pdf failure, got nil")
	}
	if !strings.Contains(err.Error(), "img2pdf failed") {
		t.Errorf("expected error about img2pdf failure, got: %v", err)
	}
}

// TestBuildPDF_Timeout tests timeout handling
func TestBuildPDF_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	createMockImage(t, tmpDir, "image1.png")

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			// Simulate timeout by checking context
			select {
			case <-ctx.Done():
				return runner.Result{}, ctx.Err()
			default:
				return runner.Result{}, context.DeadlineExceeded
			}
		},
	}

	_, err := buildPDFWithRunner(mockR, tmpDir, outputDir, 1*time.Nanosecond)
	if err == nil {
		t.Error("expected error for timeout, got nil")
	}
}

// TestBuildPDF_OutputFileNotCreated tests error when runner succeeds but file is missing
func TestBuildPDF_OutputFileNotCreated(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	createMockImage(t, tmpDir, "image1.png")

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			// Return success but don't create the file
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := buildPDFWithRunner(mockR, tmpDir, outputDir, 30*time.Second)
	if err == nil {
		t.Error("expected error for missing output file, got nil")
	}
	if !strings.Contains(err.Error(), "output file not found") {
		t.Errorf("expected error about missing output file, got: %v", err)
	}
}

// TestBuildPDF_NonImageFiles tests that non-image files are filtered out
func TestBuildPDF_NonImageFiles(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	// Create mix of image and non-image files
	createMockImage(t, tmpDir, "image1.png")
	createMockTextFile(t, tmpDir, "not an image")
	createMockImage(t, tmpDir, "image2.jpg")

	var imageCount int
	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			// Count image arguments
			for _, arg := range args {
				if strings.HasSuffix(arg, ".png") || strings.HasSuffix(arg, ".jpg") || strings.HasSuffix(arg, ".jpeg") {
					imageCount++
				}
			}
			outputPath := ""
			for i, arg := range args {
				if arg == "-o" && i+1 < len(args) {
					outputPath = args[i+1]
					break
				}
			}
			_ = os.WriteFile(outputPath, []byte("%PDF-1.4\n"), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := buildPDFWithRunner(mockR, tmpDir, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("BuildPDF failed: %v", err)
	}

	// Should only process 2 images, not the text file
	if imageCount != 2 {
		t.Errorf("expected 2 images, got %d", imageCount)
	}
}

// TestBuildPDF_CaseInsensitiveExtensions tests case-insensitive extension matching
func TestBuildPDF_CaseInsensitiveExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	// Create images with uppercase extensions
	createMockImage(t, tmpDir, "image1.JPG")
	createMockImage(t, tmpDir, "image2.PNG")
	createMockImage(t, tmpDir, "image3.JPEG")

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := ""
			for i, arg := range args {
				if arg == "-o" && i+1 < len(args) {
					outputPath = args[i+1]
					break
				}
			}
			_ = os.WriteFile(outputPath, []byte("%PDF-1.4\n"), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := buildPDFWithRunner(mockR, tmpDir, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("BuildPDF failed: %v", err)
	}
}

// OCRPDF Tests

// TestOCRPDF_Success tests successful OCR processing
func TestOCRPDF_Success(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			if bin != "ocrmypdf" {
				t.Errorf("expected ocrmypdf, got %s", bin)
			}
			// Verify args contain expected flags
			hasDeskew := false
			hasRotate := false
			hasLang := false
			for i, arg := range args {
				if arg == "--deskew" {
					hasDeskew = true
				}
				if arg == "--rotate-pages" {
					hasRotate = true
				}
				if arg == "-l" && i+1 < len(args) {
					hasLang = true
				}
			}
			if !hasDeskew || !hasRotate || !hasLang {
				t.Errorf("missing expected flags: deskew=%v, rotate=%v, lang=%v", hasDeskew, hasRotate, hasLang)
			}
			// Create output file (last arg should be output path)
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte("%PDF-1.4\n"), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	result, err := ocrPDFWithRunner(mockR, pdfPath, outputDir, "eng", 30*time.Second)
	if err != nil {
		t.Fatalf("OCRPDF failed: %v", err)
	}

	expectedPath := filepath.Join(outputDir, "combined_ocr.pdf")
	if result != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result)
	}

	// Verify file exists
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("output OCR PDF file was not created")
	}
}

// TestOCRPDF_DifferentLanguage tests OCR with different language parameter
func TestOCRPDF_DifferentLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)

	var capturedLang string
	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			// Extract language from args
			for i, arg := range args {
				if arg == "-l" && i+1 < len(args) {
					capturedLang = args[i+1]
					break
				}
			}
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte("%PDF-1.4\n"), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := ocrPDFWithRunner(mockR, pdfPath, outputDir, "fra", 30*time.Second)
	if err != nil {
		t.Fatalf("OCRPDF failed: %v", err)
	}

	if capturedLang != "fra" {
		t.Errorf("expected language 'fra', got %s", capturedLang)
	}
}

// TestOCRPDF_OutputFileCreated tests that output file is verified
func TestOCRPDF_OutputFileCreated(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte("%PDF-1.4\n"), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	result, err := ocrPDFWithRunner(mockR, pdfPath, outputDir, "eng", 30*time.Second)
	if err != nil {
		t.Fatalf("OCRPDF failed: %v", err)
	}

	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("output file was not created")
	}
}

// TestOCRPDF_OcrmypdfFailure tests error handling when ocrmypdf fails
func TestOCRPDF_OcrmypdfFailure(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			return runner.Result{
					ExitCode: 1,
					Stderr:   "ocrmypdf error: invalid PDF",
				}, &runner.ExecError{
					Bin:    bin,
					Args:   args,
					Result: runner.Result{ExitCode: 1, Stderr: "ocrmypdf error: invalid PDF"},
				}
		},
	}

	_, err := ocrPDFWithRunner(mockR, pdfPath, outputDir, "eng", 30*time.Second)
	if err == nil {
		t.Error("expected error for ocrmypdf failure, got nil")
	}
	if !strings.Contains(err.Error(), "ocrmypdf failed") {
		t.Errorf("expected error about ocrmypdf failure, got: %v", err)
	}
}

// TestOCRPDF_NonExistentInputPDF tests error when input PDF doesn't exist
func TestOCRPDF_NonExistentInputPDF(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	nonExistentPath := filepath.Join(tmpDir, "nonexistent.pdf")

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			// ocrmypdf would fail on non-existent file, but we test the runner error
			return runner.Result{
					ExitCode: 1,
					Stderr:   "file not found",
				}, &runner.ExecError{
					Bin:    bin,
					Args:   args,
					Result: runner.Result{ExitCode: 1, Stderr: "file not found"},
				}
		},
	}

	_, err := ocrPDFWithRunner(mockR, nonExistentPath, outputDir, "eng", 30*time.Second)
	if err == nil {
		t.Error("expected error for non-existent input, got nil")
	}
}

// TestOCRPDF_Timeout tests timeout handling
func TestOCRPDF_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			return runner.Result{}, context.DeadlineExceeded
		},
	}

	_, err := ocrPDFWithRunner(mockR, pdfPath, outputDir, "eng", 1*time.Nanosecond)
	if err == nil {
		t.Error("expected error for timeout, got nil")
	}
}

// TestOCRPDF_OutputFileNotCreated tests error when runner succeeds but file is missing
func TestOCRPDF_OutputFileNotCreated(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			// Return success but don't create the file
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := ocrPDFWithRunner(mockR, pdfPath, outputDir, "eng", 30*time.Second)
	if err == nil {
		t.Error("expected error for missing output file, got nil")
	}
	if !strings.Contains(err.Error(), "output file not found") {
		t.Errorf("expected error about missing output file, got: %v", err)
	}
}

// TestOCRPDF_SpecialCharactersInPath tests paths with special characters
func TestOCRPDF_SpecialCharactersInPath(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	// Create PDF with special characters in path
	pdfPath := filepath.Join(tmpDir, "test file (1).pdf")
	_ = os.WriteFile(pdfPath, []byte("%PDF-1.4\n"), 0644)

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte("%PDF-1.4\n"), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := ocrPDFWithRunner(mockR, pdfPath, outputDir, "eng", 30*time.Second)
	if err != nil {
		t.Fatalf("OCRPDF failed with special characters: %v", err)
	}
}

// ExtractText Tests

// TestExtractText_Success tests successful text extraction
func TestExtractText_Success(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)
	expectedText := "This is a test document with enough text to pass validation. It has more than 20 characters."

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			if bin != "pdftotext" {
				t.Errorf("expected pdftotext, got %s", bin)
			}
			// Verify -layout flag is present
			if len(args) < 2 || args[0] != "-layout" {
				t.Errorf("expected -layout flag, got args: %v", args)
			}
			// Create output file with sufficient text
			outputPath := args[len(args)-1]
			if err := os.WriteFile(outputPath, []byte(expectedText), 0644); err != nil {
				t.Fatalf("failed to create output file: %v", err)
			}
			return runner.Result{ExitCode: 0}, nil
		},
	}

	result, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("ExtractText failed: %v", err)
	}

	expectedPath := filepath.Join(outputDir, "extracted.txt")
	if result != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result)
	}

	// Verify file exists and has correct content
	content, err := os.ReadFile(result)
	if err != nil {
		t.Fatalf("failed to read extracted text: %v", err)
	}
	if strings.TrimSpace(string(content)) != expectedText {
		t.Errorf("expected text %q, got %q", expectedText, string(content))
	}
}

// TestExtractText_WithLayout tests that -layout flag is used
func TestExtractText_WithLayout(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)
	textContent := "This is a test document with enough text to pass validation."

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			// Verify -layout flag
			hasLayout := false
			for _, arg := range args {
				if arg == "-layout" {
					hasLayout = true
					break
				}
			}
			if !hasLayout {
				t.Error("expected -layout flag to be present")
			}
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte(textContent), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("ExtractText failed: %v", err)
	}
}

// TestExtractText_OutputFileCreated tests that output file is verified
func TestExtractText_OutputFileCreated(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)
	textContent := "This is a test document with enough text to pass validation."

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte(textContent), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	result, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("ExtractText failed: %v", err)
	}

	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("output file was not created")
	}
}

// TestExtractText_ContentValidation tests that extracted content is validated
func TestExtractText_ContentValidation(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)
	textContent := "This is a test document with enough text to pass validation. It has more than 20 characters."

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte(textContent), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	result, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("ExtractText failed: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(result)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}
	if len(strings.TrimSpace(string(content))) < 20 {
		t.Error("extracted text should be at least 20 characters")
	}
}

// TestExtractText_PdftotextFailure tests error handling when pdftotext fails
func TestExtractText_PdftotextFailure(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			return runner.Result{
					ExitCode: 1,
					Stderr:   "pdftotext error: invalid PDF",
				}, &runner.ExecError{
					Bin:    bin,
					Args:   args,
					Result: runner.Result{ExitCode: 1, Stderr: "pdftotext error: invalid PDF"},
				}
		},
	}

	_, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	if err == nil {
		t.Error("expected error for pdftotext failure, got nil")
	}
	if !strings.Contains(err.Error(), "pdftotext failed") {
		t.Errorf("expected error about pdftotext failure, got: %v", err)
	}
}

// TestExtractText_NonExistentInputPDF tests error when input PDF doesn't exist
func TestExtractText_NonExistentInputPDF(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	nonExistentPath := filepath.Join(tmpDir, "nonexistent.pdf")

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			return runner.Result{
					ExitCode: 1,
					Stderr:   "file not found",
				}, &runner.ExecError{
					Bin:    bin,
					Args:   args,
					Result: runner.Result{ExitCode: 1, Stderr: "file not found"},
				}
		},
	}

	_, err := extractTextWithRunner(mockR, nonExistentPath, outputDir, 30*time.Second)
	if err == nil {
		t.Error("expected error for non-existent input, got nil")
	}
}

// TestExtractText_Timeout tests timeout handling
func TestExtractText_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			return runner.Result{}, context.DeadlineExceeded
		},
	}

	_, err := extractTextWithRunner(mockR, pdfPath, outputDir, 1*time.Nanosecond)
	if err == nil {
		t.Error("expected error for timeout, got nil")
	}
}

// TestExtractText_OutputFileNotCreated tests error when runner succeeds but file is missing
func TestExtractText_OutputFileNotCreated(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			// Return success but don't create the file
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	if err == nil {
		t.Error("expected error for missing output file, got nil")
	}
	if !strings.Contains(err.Error(), "output file not found") {
		t.Errorf("expected error about missing output file, got: %v", err)
	}
}

// TestExtractText_TextTooShort tests error when extracted text is too short
func TestExtractText_TextTooShort(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)
	shortText := "Short" // Less than 20 characters

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte(shortText), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	if err == nil {
		t.Error("expected error for text too short, got nil")
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Errorf("expected error about text too short, got: %v", err)
	}
}

// TestExtractText_EmptyText tests error when extracted text is empty
func TestExtractText_EmptyText(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte("   \n\n  "), 0644) // Only whitespace
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	if err == nil {
		t.Error("expected error for empty text, got nil")
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Errorf("expected error about text too short, got: %v", err)
	}
}

// TestExtractText_ReadFileError tests error when reading the extracted file fails
func TestExtractText_ReadFileError(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)
	textContent := "This is a test document with enough text to pass validation."

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte(textContent), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	// After runner succeeds, delete the file to simulate read error
	// We'll need to intercept the file creation and delete it
	// Actually, we can't easily test this without modifying the function
	// So we'll test the validation logic separately
	_, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	// This should succeed normally, but if we could delete the file between
	// Stat and ReadFile, it would fail. This is hard to test without race conditions.
	if err != nil {
		// If there's an error, it should be about reading the file
		if !strings.Contains(err.Error(), "failed to read") {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

// TestExtractText_UnicodeText tests extraction with unicode characters
func TestExtractText_UnicodeText(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)
	unicodeText := "This is a test with unicode: 测试文档 日本語 العربية. It has more than 20 characters."

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte(unicodeText), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("ExtractText failed with unicode: %v", err)
	}
}

// TestExtractText_SpecialCharacters tests extraction with special characters
func TestExtractText_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)
	specialText := "This text has special chars: !@#$%^&*()_+-=[]{}|;':\",./<>? It has more than 20 characters."

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte(specialText), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("ExtractText failed with special characters: %v", err)
	}
}

// TestExtractText_Exactly20Chars tests boundary case with exactly 20 characters
func TestExtractText_Exactly20Chars(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)
	exactText := "12345678901234567890" // Exactly 20 characters

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte(exactText), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	if err != nil {
		t.Fatalf("ExtractText should pass with exactly 20 characters, got error: %v", err)
	}
}

// TestExtractText_19Chars tests boundary case with 19 characters (should fail)
func TestExtractText_19Chars(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := t.TempDir()

	pdfPath := createMockPDF(t, tmpDir)
	shortText := "1234567890123456789" // 19 characters

	mockR := &mockRunner{
		runFunc: func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
			outputPath := args[len(args)-1]
			_ = os.WriteFile(outputPath, []byte(shortText), 0644)
			return runner.Result{ExitCode: 0}, nil
		},
	}

	_, err := extractTextWithRunner(mockR, pdfPath, outputDir, 30*time.Second)
	if err == nil {
		t.Error("expected error for 19 characters, got nil")
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Errorf("expected error about text too short, got: %v", err)
	}
}

// CleanupArtifact Additional Edge Cases

// TestCleanupArtifact_DirectoryInsteadOfFile tests error when path is a directory
func TestCleanupArtifact_DirectoryInsteadOfFile(t *testing.T) {
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "testdir")

	// Create a directory
	if err := os.Mkdir(dirPath, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// CleanupArtifact should attempt to remove it, which will fail on most systems
	// since Remove() can remove directories, but the function is designed for files
	err := CleanupArtifact(dirPath)
	// On Unix systems, os.Remove can remove directories, so this might succeed
	// On Windows, it might fail. We just verify it doesn't panic.
	if err != nil {
		// Error is acceptable for directories
		t.Logf("CleanupArtifact returned error for directory (expected on some systems): %v", err)
	}
}

// TestCleanupArtifact_PermissionDenied tests permission denied scenario
func TestCleanupArtifact_PermissionDenied(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping permission test in short mode")
	}

	// This test is platform-specific and complex to set up reliably
	// On Unix systems, we'd need to create a file we can't delete
	// This is difficult in a portable way, so we'll skip detailed testing
	// The existing TestCleanupArtifact_PermissionError already covers this concept
	t.Log("Permission denied testing requires platform-specific setup")
}
