package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
