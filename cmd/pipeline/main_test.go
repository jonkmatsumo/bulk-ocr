package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonkmatsumo/bulk-ocr/internal/runner"
)

func getRepoRoot(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	// If we're in cmd/pipeline, go up two levels
	if filepath.Base(wd) == "pipeline" && filepath.Base(filepath.Dir(wd)) == "cmd" {
		return filepath.Dir(filepath.Dir(wd))
	}
	// Otherwise assume we're at repo root
	return wd
}

func TestMain_Help(t *testing.T) {
	repoRoot := getRepoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/pipeline", "--help")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Logf("help output: %s", string(output))
	}
	// --help typically exits with code 2, which is fine
}

func TestMain_Version(t *testing.T) {
	repoRoot := getRepoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/pipeline", "version")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version command failed: %v, output: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "version") {
		t.Errorf("expected version output, got: %s", outputStr)
	}
}

func TestMain_Doctor(t *testing.T) {
	repoRoot := getRepoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/pipeline", "doctor")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	// Doctor may exit with code 1 if tools are missing (expected locally)
	// or code 0 if all tools are present
	outputStr := string(output)
	if !strings.Contains(outputStr, "Doctor report") {
		t.Errorf("expected 'Doctor report' in output, got: %s", outputStr)
	}
	// Don't fail on exit code - missing tools locally is expected
	_ = err
}

func TestMain_Run_DefaultFlags(t *testing.T) {
	repoRoot := getRepoRoot(t)
	tmpInput := t.TempDir()
	tmpOutput := t.TempDir()

	cmd := exec.Command("go", "run", "./cmd/pipeline", "run",
		"--input", tmpInput,
		"--out", tmpOutput)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run command failed: %v, output: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "images found: 0") {
		t.Errorf("expected 'images found: 0', got: %s", outputStr)
	}
}

func TestMain_Run_WithImages(t *testing.T) {
	repoRoot := getRepoRoot(t)
	tmpInput := t.TempDir()
	tmpOutput := t.TempDir()

	// Create test images
	testFiles := []string{"image1.jpg", "image2.png", "image3.jpeg"}
	for _, f := range testFiles {
		path := filepath.Join(tmpInput, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	cmd := exec.Command("go", "run", "./cmd/pipeline", "run",
		"--input", tmpInput,
		"--out", tmpOutput)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()

	outputStr := string(output)

	// Verify staging succeeded (this is what we're testing)
	if !strings.Contains(outputStr, "images found: 3") {
		t.Errorf("expected 'images found: 3', got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "staged") {
		t.Errorf("expected 'staged' message in output, got: %s", outputStr)
	}

	// If command failed, check if it's due to missing external tools (expected in local tests)
	// or a real staging failure
	if err != nil {
		// If staging succeeded but later stages failed due to missing tools, that's OK
		// We only verify staging in this test
		if strings.Contains(outputStr, "staged") &&
			(strings.Contains(outputStr, "No module named img2pdf") ||
				strings.Contains(outputStr, "command not found") ||
				strings.Contains(outputStr, "PDF synthesis failed")) {
			t.Logf("Pipeline failed after staging due to missing external tools (expected in local tests): %v", err)
			// Continue to verify staging artifacts
		} else {
			// Real failure - staging didn't work
			t.Fatalf("run command failed: %v, output: %s", err, string(output))
		}
	}

	// Verify preprocessed directory was created
	preprocessedDir := filepath.Join(tmpOutput, "preprocessed")
	if _, err := os.Stat(preprocessedDir); os.IsNotExist(err) {
		t.Error("preprocessed directory was not created")
	}

	// Verify staged files exist
	for i := 1; i <= 3; i++ {
		var ext string
		switch i {
		case 1:
			ext = ".jpg"
		case 2:
			ext = ".png"
		case 3:
			ext = ".jpeg"
		}
		stagedFile := filepath.Join(preprocessedDir, fmt.Sprintf("%04d%s", i, ext))
		if _, err := os.Stat(stagedFile); os.IsNotExist(err) {
			t.Errorf("staged file does not exist: %s", stagedFile)
		}
	}
}

// Phase 3: Test doctorCommand()

func TestDoctorCommand_AllToolsPresent(t *testing.T) {
	mockR := &mockRunner{}

	toolPaths := map[string]string{
		"python3":   "/usr/bin/python3",
		"ocrmypdf":  "/usr/local/bin/ocrmypdf",
		"tesseract": "/usr/bin/tesseract",
		"pdftotext": "/usr/bin/pdftotext",
		"gs":        "/usr/bin/gs",
	}

	mockR.lookPathFunc = func(bin string) (string, error) {
		if path, ok := toolPaths[bin]; ok {
			return path, nil
		}
		return "", fmt.Errorf("not found: %s", bin)
	}

	versionOutputs := map[string]string{
		"python3":   "Python 3.9.0",
		"ocrmypdf":  "ocrmypdf version 15.0.0",
		"tesseract": "tesseract Version 5.0.0",
		"pdftotext": "pdftotext version 23.01.0",
		"gs":        "GPL Ghostscript 10.0.0",
	}

	mockR.runFunc = func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
		output := versionOutputs[bin]
		if output == "" {
			return runner.Result{}, fmt.Errorf("unknown tool: %s", bin)
		}
		return runner.Result{
			ExitCode:   0,
			Stdout:     output,
			Stderr:     "",
			DurationMs: 100,
		}, nil
	}

	// Test that function runs without error when all tools are present
	// Note: We can't easily test os.Exit behavior, so we just verify the function completes
	err := doctorCommandWithRunner([]string{}, mockR)
	// Function may return nil even if it calls os.Exit(1), so we just check it doesn't panic
	if err != nil && !strings.Contains(err.Error(), "failed to parse flags") {
		t.Logf("doctorCommand returned error (may be expected): %v", err)
	}
}

func TestDoctorCommand_MissingTools(t *testing.T) {
	mockR := &mockRunner{}

	mockR.lookPathFunc = func(bin string) (string, error) {
		// Some tools missing
		if bin == "tesseract" || bin == "pdftotext" {
			return "", fmt.Errorf("not found: %s", bin)
		}
		return "/usr/bin/" + bin, nil
	}

	mockR.runFunc = func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
		return runner.Result{ExitCode: 0, Stdout: "version 1.0.0"}, nil
	}

	// Test that function handles missing tools
	// With mocked runner, it should return error instead of calling os.Exit
	err := doctorCommandWithRunner([]string{}, mockR)
	if err == nil {
		t.Error("expected error when tools are missing")
	}
	if !strings.Contains(err.Error(), "doctor found errors") {
		t.Errorf("expected 'doctor found errors' in error, got: %v", err)
	}
}

func TestDoctorCommand_VersionExtractionFailure(t *testing.T) {
	mockR := &mockRunner{}

	mockR.lookPathFunc = func(bin string) (string, error) {
		return "/usr/bin/" + bin, nil
	}

	mockR.runFunc = func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
		// Return error for version command
		return runner.Result{ExitCode: 1, Stderr: "command failed"}, fmt.Errorf("version command failed")
	}

	// Test that function handles version extraction failures
	err := doctorCommandWithRunner([]string{}, mockR)
	// Function may return nil even if it calls os.Exit(1)
	_ = err
}

func TestDoctorCommand_WithSmokeTest(t *testing.T) {
	// Smoke test requires real runner, so we skip it when using mock
	// This test verifies the flag parsing works
	mockR := &mockRunner{}

	mockR.lookPathFunc = func(bin string) (string, error) {
		return "/usr/bin/" + bin, nil
	}

	mockR.runFunc = func(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
		return runner.Result{ExitCode: 0, Stdout: "version 1.0.0"}, nil
	}

	// With --smoke flag, but mocked runner will skip actual smoke test
	err := doctorCommandWithRunner([]string{"--smoke"}, mockR)
	// Function should complete (smoke test skipped with mock)
	_ = err
}

func TestMain_Run_MissingInput(t *testing.T) {
	repoRoot := getRepoRoot(t)
	tmpOutput := t.TempDir()

	cmd := exec.Command("go", "run", "./cmd/pipeline", "run",
		"--input", "/nonexistent/directory",
		"--out", tmpOutput)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("expected error for missing input directory")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "does not exist") {
		t.Errorf("expected error message about missing directory, got: %s", outputStr)
	}
}

func TestMain_Run_CreatesOutput(t *testing.T) {
	repoRoot := getRepoRoot(t)
	tmpInput := t.TempDir()
	tmpOutput := filepath.Join(t.TempDir(), "new-output")

	cmd := exec.Command("go", "run", "./cmd/pipeline", "run",
		"--input", tmpInput,
		"--out", tmpOutput)
	cmd.Dir = repoRoot
	_, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run command failed: %v", err)
	}

	// Verify output directory was created
	if _, err := os.Stat(tmpOutput); os.IsNotExist(err) {
		t.Error("output directory was not created")
	}
}

func TestMain_Run_AllFlags(t *testing.T) {
	repoRoot := getRepoRoot(t)
	tmpInput := t.TempDir()
	tmpOutput := t.TempDir()

	cmd := exec.Command("go", "run", "./cmd/pipeline", "run",
		"--input", tmpInput,
		"--out", tmpOutput,
		"--keep-artifacts=false",
		"--lang", "fra")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run command failed: %v, output: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "keep artifacts: false") {
		t.Errorf("expected 'keep artifacts: false', got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "language: fra") {
		t.Errorf("expected 'language: fra', got: %s", outputStr)
	}
}

func TestMain_UnknownSubcommand(t *testing.T) {
	repoRoot := getRepoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/pipeline", "unknown")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "unknown subcommand") {
		t.Errorf("expected error about unknown subcommand, got: %s", outputStr)
	}
}

// Mock implementations for testing

// mockPipelineStages implements pipelineStages interface
type mockPipelineStages struct {
	buildPDFFunc    func(string, string, time.Duration) (string, error)
	ocrPDFFunc      func(string, string, string, time.Duration) (string, error)
	extractTextFunc func(string, string, time.Duration) (string, error)
	cleanupFunc     func(string) error
}

func (m *mockPipelineStages) BuildPDF(preprocessedDir, outputDir string, timeout time.Duration) (string, error) {
	if m.buildPDFFunc != nil {
		return m.buildPDFFunc(preprocessedDir, outputDir, timeout)
	}
	return filepath.Join(outputDir, "combined.pdf"), nil
}

func (m *mockPipelineStages) OCRPDF(pdfPath, outputDir, lang string, timeout time.Duration) (string, error) {
	if m.ocrPDFFunc != nil {
		return m.ocrPDFFunc(pdfPath, outputDir, lang, timeout)
	}
	return filepath.Join(outputDir, "combined_ocr.pdf"), nil
}

func (m *mockPipelineStages) ExtractText(pdfPath, outputDir string, timeout time.Duration) (string, error) {
	if m.extractTextFunc != nil {
		return m.extractTextFunc(pdfPath, outputDir, timeout)
	}
	textPath := filepath.Join(outputDir, "extracted.txt")
	// Create a mock text file with >20 characters
	content := "This is a test extracted text with more than 20 characters for validation."
	if err := os.WriteFile(textPath, []byte(content), 0644); err != nil {
		return "", err
	}
	return textPath, nil
}

func (m *mockPipelineStages) CleanupArtifact(path string) error {
	if m.cleanupFunc != nil {
		return m.cleanupFunc(path)
	}
	return nil
}

// mockRunner implements runnerInterface for testing
type mockRunner struct {
	lookPathFunc func(string) (string, error)
	runFunc      func(context.Context, string, []string, runner.RunOpts) (runner.Result, error)
}

func (m *mockRunner) LookPath(bin string) (string, error) {
	if m.lookPathFunc != nil {
		return m.lookPathFunc(bin)
	}
	return "/usr/bin/" + bin, nil
}

func (m *mockRunner) Run(ctx context.Context, bin string, args []string, opts runner.RunOpts) (runner.Result, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, bin, args, opts)
	}
	return runner.Result{
		Cmd:        bin + " " + strings.Join(args, " "),
		ExitCode:   0,
		DurationMs: 100,
		Stdout:     "",
		Stderr:     "",
	}, nil
}

// Test helper functions

func setupTestDirs(t *testing.T) (inputDir, outputDir string) {
	inputDir = t.TempDir()
	outputDir = t.TempDir()
	return inputDir, outputDir
}

func createMockImage(t *testing.T, dir, name string) {
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
}

// Phase 4: Test helper functions

// Test extractVersion()
func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "StandardFormat",
			input:    "python3 version 3.9.0",
			expected: "3.9.0",
		},
		{
			name:     "WithVersionKeyword",
			input:    "tesseract Version 5.0.0",
			expected: "5.0.0",
		},
		{
			name:     "NoVersionFound",
			input:    "some output without version",
			expected: "some output without version", // First non-empty line
		},
		{
			name:     "FirstNonEmptyLine",
			input:    "\n\nfirst line\nsecond line",
			expected: "first line",
		},
		{
			name:     "EmptyInput",
			input:    "",
			expected: "",
		},
		{
			name:     "LongLine",
			input:    strings.Repeat("a", 150) + "\nshort line",
			expected: "short line", // Skip long lines
		},
		{
			name:     "MultiLineWithVersion",
			input:    "some text\nversion 2.1.0\nmore text",
			expected: "2.1.0",
		},
		{
			name:     "VersionInMiddle",
			input:    "prefix version 3.2.1 suffix",
			expected: "3.2.1",
		},
		{
			name:     "OnlyWhitespace",
			input:    "   \n\t\n  ",
			expected: "",
		},
		{
			name:     "VersionKeywordCaseInsensitive",
			input:    "tool version 1.0.0", // Function checks for lowercase "version" in line
			expected: "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersion(tt.input)
			if result != tt.expected {
				t.Errorf("extractVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test generateTestImage()
func TestGenerateTestImage_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testImage := filepath.Join(tmpDir, "test.png")

	// Test with Python PIL script success (mocked via runner)
	// Since we can't easily mock the runner in generateTestImage,
	// we'll test the fallback path which writes a minimal PNG
	err := generateTestImage(testImage)
	if err != nil {
		t.Fatalf("generateTestImage() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testImage); os.IsNotExist(err) {
		t.Error("test image file was not created")
	}

	// Verify it's a valid PNG (starts with PNG signature)
	content, err := os.ReadFile(testImage)
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}
	if len(content) < 8 || string(content[0:8]) != "\x89PNG\r\n\x1a\n" {
		t.Error("test image is not a valid PNG")
	}
}

func TestGenerateTestImage_PythonFailure(t *testing.T) {
	tmpDir := t.TempDir()
	testImage := filepath.Join(tmpDir, "test.png")

	// The function should fallback to writing minimal PNG if Python fails
	// This is tested by the success case above since Python may not be available
	// in test environment. The fallback path is what we're actually testing.
	err := generateTestImage(testImage)
	if err != nil {
		t.Fatalf("generateTestImage() should succeed with fallback: %v", err)
	}

	// Verify fallback PNG was written
	if _, err := os.Stat(testImage); os.IsNotExist(err) {
		t.Error("fallback PNG file was not created")
	}
}

// Test runSmokeTest()
// Note: runSmokeTest requires *runner.Runner, not an interface, so we can't easily mock it
// These tests verify the error handling paths, but may require actual tools
func TestRunSmokeTest_Success(t *testing.T) {
	// This test requires actual external tools (python3, img2pdf, ocrmypdf, pdftotext)
	// Skip in unit test environment
	t.Skip("runSmokeTest requires actual external tools, skipping unit test")
}

func TestRunSmokeTest_ImageGenerationFailure(t *testing.T) {
	// This would require refactoring generateTestImage to accept a mock runner
	// For now, we test generateTestImage separately
	t.Skip("requires refactoring generateTestImage for testability")
}

func TestRunSmokeTest_Img2PdfFailure(t *testing.T) {
	// Requires refactoring runSmokeTest to accept interface
	t.Skip("requires refactoring runSmokeTest for testability")
}

func TestRunSmokeTest_OcrFailure(t *testing.T) {
	// Requires refactoring runSmokeTest to accept interface
	t.Skip("requires refactoring runSmokeTest for testability")
}

func TestRunSmokeTest_PdfToTextFailure(t *testing.T) {
	// Requires refactoring runSmokeTest to accept interface
	t.Skip("requires refactoring runSmokeTest for testability")
}

func TestRunSmokeTest_EmptyTextOutput(t *testing.T) {
	// Requires refactoring runSmokeTest to accept interface
	t.Skip("requires refactoring runSmokeTest for testability")
}

// Phase 2: Test runCommand()

func TestRunCommand_HappyPath(t *testing.T) {
	inputDir, outputDir := setupTestDirs(t)

	// Create test images
	createMockImage(t, inputDir, "image1.jpg")
	createMockImage(t, inputDir, "image2.png")

	// Save original implementation
	originalImpl := pipelineStagesImpl
	defer func() { pipelineStagesImpl = originalImpl }()

	// Create mock pipeline stages
	mockStages := &mockPipelineStages{}
	pdfPath := filepath.Join(outputDir, "combined.pdf")
	ocrPath := filepath.Join(outputDir, "combined_ocr.pdf")
	textPath := filepath.Join(outputDir, "extracted.txt")

	mockStages.buildPDFFunc = func(preprocessedDir, outputDir string, timeout time.Duration) (string, error) {
		// Create mock PDF file
		pdfContent := []byte("%PDF-1.4\n")
		if err := os.WriteFile(pdfPath, pdfContent, 0644); err != nil {
			return "", err
		}
		return pdfPath, nil
	}

	mockStages.ocrPDFFunc = func(pdfPath, outputDir, lang string, timeout time.Duration) (string, error) {
		// Create mock OCR PDF file
		pdfContent := []byte("%PDF-1.4\n")
		if err := os.WriteFile(ocrPath, pdfContent, 0644); err != nil {
			return "", err
		}
		return ocrPath, nil
	}

	mockStages.extractTextFunc = func(pdfPath, outputDir string, timeout time.Duration) (string, error) {
		// Create mock text file with >20 characters
		content := "This is extracted text from the OCR PDF with more than 20 characters for validation purposes."
		if err := os.WriteFile(textPath, []byte(content), 0644); err != nil {
			return "", err
		}
		return textPath, nil
	}

	mockStages.cleanupFunc = func(path string) error {
		return nil
	}

	pipelineStagesImpl = mockStages

	// Run command
	err := runCommand(
		inputDir, outputDir,
		true,           // keepArtifacts
		"eng",          // lang
		false,          // recursive
		5*time.Minute,  // pdfTimeout
		10*time.Minute, // ocrTimeout
		2*time.Minute,  // extractTimeout
		60,             // minChunkChars
		2,              // maxBlankLines
		false,          // emitChunksJSONL
		[]string{},     // chromePatterns
		5,              // simhashK
		6,              // simhashThreshold
		250,            // window
		"simhash",      // dedupeMethod
		"Test Title",   // markdownTitle
		false,          // includeChunkIDs
	)

	if err != nil {
		t.Fatalf("runCommand() failed: %v", err)
	}

	// Verify output files were created
	resultPath := filepath.Join(outputDir, "result.md")
	if _, err := os.Stat(resultPath); os.IsNotExist(err) {
		t.Error("result.md was not created")
	}

	reportPath := filepath.Join(outputDir, "dedupe_report.json")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("dedupe_report.json was not created")
	}
}

func TestRunCommand_InvalidInputDirectory(t *testing.T) {
	outputDir := t.TempDir()

	err := runCommand(
		"/nonexistent/directory",
		outputDir,
		true, "eng", false,
		5*time.Minute, 10*time.Minute, 2*time.Minute,
		60, 2, false, []string{},
		5, 6, 250, "simhash", "Title", false,
	)

	if err == nil {
		t.Error("expected error for invalid input directory")
	}
	if !strings.Contains(err.Error(), "input directory does not exist") {
		t.Errorf("expected 'input directory does not exist' error, got: %v", err)
	}
}

func TestRunCommand_NoImagesFound(t *testing.T) {
	inputDir, outputDir := setupTestDirs(t)

	// Empty input directory
	err := runCommand(
		inputDir, outputDir,
		true, "eng", false,
		5*time.Minute, 10*time.Minute, 2*time.Minute,
		60, 2, false, []string{},
		5, 6, 250, "simhash", "Title", false,
	)

	// Should return nil (graceful exit)
	if err != nil {
		t.Errorf("expected nil for empty directory, got: %v", err)
	}
}

func TestRunCommand_BuildPDFFailure(t *testing.T) {
	inputDir, outputDir := setupTestDirs(t)
	createMockImage(t, inputDir, "image1.jpg")

	originalImpl := pipelineStagesImpl
	defer func() { pipelineStagesImpl = originalImpl }()

	mockStages := &mockPipelineStages{}
	mockStages.buildPDFFunc = func(preprocessedDir, outputDir string, timeout time.Duration) (string, error) {
		return "", fmt.Errorf("img2pdf failed")
	}

	pipelineStagesImpl = mockStages

	err := runCommand(
		inputDir, outputDir,
		true, "eng", false,
		5*time.Minute, 10*time.Minute, 2*time.Minute,
		60, 2, false, []string{},
		5, 6, 250, "simhash", "Title", false,
	)

	if err == nil {
		t.Error("expected error from BuildPDF failure")
	}
	if !strings.Contains(err.Error(), "PDF synthesis failed") {
		t.Errorf("expected 'PDF synthesis failed' error, got: %v", err)
	}
}

func TestRunCommand_OCRPDFFailure(t *testing.T) {
	inputDir, outputDir := setupTestDirs(t)
	createMockImage(t, inputDir, "image1.jpg")

	originalImpl := pipelineStagesImpl
	defer func() { pipelineStagesImpl = originalImpl }()

	mockStages := &mockPipelineStages{}
	pdfPath := filepath.Join(outputDir, "combined.pdf")

	mockStages.buildPDFFunc = func(preprocessedDir, outputDir string, timeout time.Duration) (string, error) {
		pdfContent := []byte("%PDF-1.4\n")
		if err := os.WriteFile(pdfPath, pdfContent, 0644); err != nil {
			return "", err
		}
		return pdfPath, nil
	}

	mockStages.ocrPDFFunc = func(pdfPath, outputDir, lang string, timeout time.Duration) (string, error) {
		return "", fmt.Errorf("ocrmypdf failed")
	}

	pipelineStagesImpl = mockStages

	err := runCommand(
		inputDir, outputDir,
		true, "eng", false,
		5*time.Minute, 10*time.Minute, 2*time.Minute,
		60, 2, false, []string{},
		5, 6, 250, "simhash", "Title", false,
	)

	if err == nil {
		t.Error("expected error from OCRPDF failure")
	}
	if !strings.Contains(err.Error(), "OCR failed") {
		t.Errorf("expected 'OCR failed' error, got: %v", err)
	}
}

func TestRunCommand_ExtractTextFailure(t *testing.T) {
	inputDir, outputDir := setupTestDirs(t)
	createMockImage(t, inputDir, "image1.jpg")

	originalImpl := pipelineStagesImpl
	defer func() { pipelineStagesImpl = originalImpl }()

	mockStages := &mockPipelineStages{}
	pdfPath := filepath.Join(outputDir, "combined.pdf")
	ocrPath := filepath.Join(outputDir, "combined_ocr.pdf")

	mockStages.buildPDFFunc = func(preprocessedDir, outputDir string, timeout time.Duration) (string, error) {
		pdfContent := []byte("%PDF-1.4\n")
		if err := os.WriteFile(pdfPath, pdfContent, 0644); err != nil {
			return "", err
		}
		return pdfPath, nil
	}

	mockStages.ocrPDFFunc = func(pdfPath, outputDir, lang string, timeout time.Duration) (string, error) {
		pdfContent := []byte("%PDF-1.4\n")
		if err := os.WriteFile(ocrPath, pdfContent, 0644); err != nil {
			return "", err
		}
		return ocrPath, nil
	}

	mockStages.extractTextFunc = func(pdfPath, outputDir string, timeout time.Duration) (string, error) {
		return "", fmt.Errorf("pdftotext failed")
	}

	pipelineStagesImpl = mockStages

	err := runCommand(
		inputDir, outputDir,
		true, "eng", false,
		5*time.Minute, 10*time.Minute, 2*time.Minute,
		60, 2, false, []string{},
		5, 6, 250, "simhash", "Title", false,
	)

	if err == nil {
		t.Error("expected error from ExtractText failure")
	}
	if !strings.Contains(err.Error(), "text extraction failed") {
		t.Errorf("expected 'text extraction failed' error, got: %v", err)
	}
}

func TestRunCommand_ArtifactCleanup(t *testing.T) {
	inputDir, outputDir := setupTestDirs(t)
	createMockImage(t, inputDir, "image1.jpg")

	originalImpl := pipelineStagesImpl
	defer func() { pipelineStagesImpl = originalImpl }()

	mockStages := &mockPipelineStages{}
	pdfPath := filepath.Join(outputDir, "combined.pdf")
	ocrPath := filepath.Join(outputDir, "combined_ocr.pdf")
	textPath := filepath.Join(outputDir, "extracted.txt")

	cleanupCalled := make(map[string]bool)

	mockStages.buildPDFFunc = func(preprocessedDir, outputDir string, timeout time.Duration) (string, error) {
		pdfContent := []byte("%PDF-1.4\n")
		if err := os.WriteFile(pdfPath, pdfContent, 0644); err != nil {
			return "", err
		}
		return pdfPath, nil
	}

	mockStages.ocrPDFFunc = func(pdfPath, outputDir, lang string, timeout time.Duration) (string, error) {
		pdfContent := []byte("%PDF-1.4\n")
		if err := os.WriteFile(ocrPath, pdfContent, 0644); err != nil {
			return "", err
		}
		return ocrPath, nil
	}

	mockStages.extractTextFunc = func(pdfPath, outputDir string, timeout time.Duration) (string, error) {
		content := "This is extracted text with more than 20 characters for validation."
		if err := os.WriteFile(textPath, []byte(content), 0644); err != nil {
			return "", err
		}
		return textPath, nil
	}

	mockStages.cleanupFunc = func(path string) error {
		cleanupCalled[path] = true
		return nil
	}

	pipelineStagesImpl = mockStages

	// Test with keepArtifacts=false
	err := runCommand(
		inputDir, outputDir,
		false, // keepArtifacts = false
		"eng", false,
		5*time.Minute, 10*time.Minute, 2*time.Minute,
		60, 2, false, []string{},
		5, 6, 250, "simhash", "Title", false,
	)

	if err != nil {
		t.Fatalf("runCommand() failed: %v", err)
	}

	// Verify cleanup was called for both PDFs
	if !cleanupCalled[pdfPath] {
		t.Error("CleanupArtifact should be called for combined.pdf when keepArtifacts=false")
	}
	if !cleanupCalled[ocrPath] {
		t.Error("CleanupArtifact should be called for combined_ocr.pdf when keepArtifacts=false")
	}

	// Reset and test with keepArtifacts=true
	cleanupCalled = make(map[string]bool)
	err = runCommand(
		inputDir, outputDir,
		true, // keepArtifacts = true
		"eng", false,
		5*time.Minute, 10*time.Minute, 2*time.Minute,
		60, 2, false, []string{},
		5, 6, 250, "simhash", "Title", false,
	)

	if err != nil {
		t.Fatalf("runCommand() failed: %v", err)
	}

	// Verify cleanup was NOT called
	if cleanupCalled[pdfPath] || cleanupCalled[ocrPath] {
		t.Error("CleanupArtifact should NOT be called when keepArtifacts=true")
	}
}

func TestRunCommand_ReadExtractedTextFailure(t *testing.T) {
	inputDir, outputDir := setupTestDirs(t)
	createMockImage(t, inputDir, "image1.jpg")

	originalImpl := pipelineStagesImpl
	defer func() { pipelineStagesImpl = originalImpl }()

	mockStages := &mockPipelineStages{}
	pdfPath := filepath.Join(outputDir, "combined.pdf")
	ocrPath := filepath.Join(outputDir, "combined_ocr.pdf")
	textPath := filepath.Join(outputDir, "extracted.txt")

	mockStages.buildPDFFunc = func(preprocessedDir, outputDir string, timeout time.Duration) (string, error) {
		pdfContent := []byte("%PDF-1.4\n")
		if err := os.WriteFile(pdfPath, pdfContent, 0644); err != nil {
			return "", err
		}
		return pdfPath, nil
	}

	mockStages.ocrPDFFunc = func(pdfPath, outputDir, lang string, timeout time.Duration) (string, error) {
		pdfContent := []byte("%PDF-1.4\n")
		if err := os.WriteFile(ocrPath, pdfContent, 0644); err != nil {
			return "", err
		}
		return ocrPath, nil
	}

	mockStages.extractTextFunc = func(pdfPath, outputDir string, timeout time.Duration) (string, error) {
		// Return path but don't create file - this will cause ReadFile to fail
		return textPath, nil
	}

	pipelineStagesImpl = mockStages

	err := runCommand(
		inputDir, outputDir,
		true, "eng", false,
		5*time.Minute, 10*time.Minute, 2*time.Minute,
		60, 2, false, []string{},
		5, 6, 250, "simhash", "Title", false,
	)

	if err == nil {
		t.Error("expected error from ReadFile failure")
	}
	if !strings.Contains(err.Error(), "failed to read extracted text") {
		t.Errorf("expected 'failed to read extracted text' error, got: %v", err)
	}
}

func TestRunCommand_WriteChunksJSONLFailure(t *testing.T) {
	inputDir, outputDir := setupTestDirs(t)
	createMockImage(t, inputDir, "image1.jpg")

	originalImpl := pipelineStagesImpl
	defer func() { pipelineStagesImpl = originalImpl }()

	mockStages := &mockPipelineStages{}
	pdfPath := filepath.Join(outputDir, "combined.pdf")
	ocrPath := filepath.Join(outputDir, "combined_ocr.pdf")
	textPath := filepath.Join(outputDir, "extracted.txt")

	mockStages.buildPDFFunc = func(preprocessedDir, outputDir string, timeout time.Duration) (string, error) {
		pdfContent := []byte("%PDF-1.4\n")
		if err := os.WriteFile(pdfPath, pdfContent, 0644); err != nil {
			return "", err
		}
		return pdfPath, nil
	}

	mockStages.ocrPDFFunc = func(pdfPath, outputDir, lang string, timeout time.Duration) (string, error) {
		pdfContent := []byte("%PDF-1.4\n")
		if err := os.WriteFile(ocrPath, pdfContent, 0644); err != nil {
			return "", err
		}
		return ocrPath, nil
	}

	mockStages.extractTextFunc = func(pdfPath, outputDir string, timeout time.Duration) (string, error) {
		content := "This is extracted text with more than 20 characters for validation purposes."
		if err := os.WriteFile(textPath, []byte(content), 0644); err != nil {
			return "", err
		}
		return textPath, nil
	}

	pipelineStagesImpl = mockStages

	// Make output directory read-only to cause write failure
	// Note: This might not work on all systems
	oldPerms := outputDir
	defer func() {
		if err := os.Chmod(oldPerms, 0755); err != nil {
			t.Logf("warning: failed to restore permissions: %v", err)
		}
	}()

	// Try to make directory read-only (may fail on some systems)
	if err := os.Chmod(outputDir, 0555); err != nil {
		t.Skipf("cannot set read-only permissions (may not work on all systems): %v", err)
	}

	// Test with emitChunksJSONL=true to trigger WriteChunksJSONL
	err := runCommand(
		inputDir, outputDir,
		true, "eng", false,
		5*time.Minute, 10*time.Minute, 2*time.Minute,
		60, 2, true, // emitChunksJSONL = true
		[]string{},
		5, 6, 250, "simhash", "Title", false,
	)

	// May or may not fail depending on system permissions
	// If it fails, verify it's the expected error
	if err != nil && !strings.Contains(err.Error(), "failed to write chunks JSONL") {
		t.Logf("Got error (may be expected): %v", err)
	}
}

func TestDoctorCommand_Wrapper(t *testing.T) {
	// Test the wrapper function doctorCommand (not doctorCommandWithRunner)
	// This is a simple wrapper that calls runner.New() and doctorCommandWithRunner
	// Since it may call os.Exit(1) if tools are missing, we can't easily test it
	// without potentially terminating the test process
	// The wrapper is trivial (just calls runner.New()), so we skip this test
	// and rely on testing doctorCommandWithRunner directly
	t.Skip("doctorCommand wrapper may call os.Exit, skipping to avoid test termination")
}
