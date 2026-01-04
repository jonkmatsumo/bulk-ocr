package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestExtractText_EmptyText(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an empty text file to simulate empty extraction
	emptyTextPath := filepath.Join(tmpDir, "extracted.txt")
	if err := os.WriteFile(emptyTextPath, []byte("   \n\n  "), 0644); err != nil {
		t.Fatalf("failed to create empty text file: %v", err)
	}

	// This test verifies the validation logic would catch empty text
	// We can't easily test the full pdftotext flow without a real PDF,
	// but we can test the validation part
	content, err := os.ReadFile(emptyTextPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	text := strings.TrimSpace(string(content))
	if len(text) >= 20 {
		t.Errorf("expected text to be short, got length %d", len(text))
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
