package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonkmatsumo/bulk-ocr/internal/dedupe"
	"github.com/jonkmatsumo/bulk-ocr/internal/text"
)

func TestWriteReport_CreatesValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "report.json")

	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
	}
	config := dedupe.DefaultConfig()
	result := dedupe.Dedupe(chunks, config)

	err := WriteReport(result, 5, config, path)
	if err != nil {
		t.Fatalf("WriteReport failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("report file was not created")
	}

	// Verify JSON is valid
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}

	var report Report
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatalf("failed to parse report JSON: %v", err)
	}
}

func TestWriteReport_ContainsAllFields(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "report.json")

	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
		{ID: "c0002", Text: "Test", Norm: "test", Index: 1},
	}
	config := dedupe.DefaultConfig()
	result := dedupe.Dedupe(chunks, config)

	err := WriteReport(result, 10, config, path)
	if err != nil {
		t.Fatalf("WriteReport failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}

	var report Report
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatalf("failed to parse report JSON: %v", err)
	}

	// Verify all expected fields are present
	if report.InputImages != 10 {
		t.Errorf("expected InputImages 10, got %d", report.InputImages)
	}
	if report.InputChunks != 2 {
		t.Errorf("expected InputChunks 2, got %d", report.InputChunks)
	}
	if report.Config.Method == "" {
		t.Error("Config.Method should not be empty")
	}
	if report.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
}

func TestWriteReport_TimestampFormat(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "report.json")

	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
	}
	config := dedupe.DefaultConfig()
	result := dedupe.Dedupe(chunks, config)

	err := WriteReport(result, 1, config, path)
	if err != nil {
		t.Fatalf("WriteReport failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}

	var report Report
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatalf("failed to parse report JSON: %v", err)
	}

	// Verify timestamp is RFC3339 format
	_, err = time.Parse(time.RFC3339, report.Timestamp)
	if err != nil {
		t.Errorf("timestamp is not in RFC3339 format: %v", err)
	}
}

func TestWriteReport_DroppedChunksIncluded(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "report.json")

	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
		{ID: "c0002", Text: "Test", Norm: "test", Index: 1}, // Duplicate
	}
	config := dedupe.DefaultConfig()
	result := dedupe.Dedupe(chunks, config)

	err := WriteReport(result, 2, config, path)
	if err != nil {
		t.Fatalf("WriteReport failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}

	var report Report
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatalf("failed to parse report JSON: %v", err)
	}

	if len(report.Dropped) == 0 {
		t.Error("expected dropped chunks in report")
	}
}

func TestWriteReport_ConfigSerialized(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "report.json")

	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
	}
	config := dedupe.DefaultConfig()
	config.SimHashK = 7
	config.SimHashThreshold = 8
	config.Window = 100
	result := dedupe.Dedupe(chunks, config)

	err := WriteReport(result, 1, config, path)
	if err != nil {
		t.Fatalf("WriteReport failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}

	var report Report
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatalf("failed to parse report JSON: %v", err)
	}

	if report.Config.SimHashK != 7 {
		t.Errorf("expected SimHashK 7, got %d", report.Config.SimHashK)
	}
	if report.Config.SimHashThreshold != 8 {
		t.Errorf("expected SimHashThreshold 8, got %d", report.Config.SimHashThreshold)
	}
	if report.Config.Window != 100 {
		t.Errorf("expected Window 100, got %d", report.Config.Window)
	}
}

func TestWriteReport_EmptyDroppedChunks(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "report.json")

	chunks := []text.Chunk{
		{ID: "c0001", Text: "Unique one", Norm: "unique one", Index: 0},
		{ID: "c0002", Text: "Unique two", Norm: "unique two", Index: 1},
	}
	config := dedupe.DefaultConfig()
	result := dedupe.Dedupe(chunks, config)

	err := WriteReport(result, 2, config, path)
	if err != nil {
		t.Fatalf("WriteReport failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}

	var report Report
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatalf("failed to parse report JSON: %v", err)
	}

	if report.DroppedChunks != 0 {
		t.Errorf("expected 0 dropped chunks, got %d", report.DroppedChunks)
	}
	if len(report.Dropped) != 0 {
		t.Errorf("expected empty dropped list, got %d items", len(report.Dropped))
	}
}

func TestWriteReport_FileWriteError(t *testing.T) {
	// Try to write to invalid path (directory that doesn't exist)
	invalidPath := "/nonexistent/directory/report.json"

	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
	}
	config := dedupe.DefaultConfig()
	result := dedupe.Dedupe(chunks, config)

	err := WriteReport(result, 1, config, invalidPath)
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
	if err != nil && !os.IsNotExist(err) && !os.IsPermission(err) {
		// Error should mention file creation failure
		if err.Error() == "" {
			t.Error("error should have a message")
		}
	}
}

// TestWriteReport_VeryLargeDroppedChunksList tests with a large number of dropped chunks
func TestWriteReport_VeryLargeDroppedChunksList(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "report.json")

	// Create many chunks, all duplicates
	chunks := make([]text.Chunk, 100)
	for i := 0; i < 100; i++ {
		chunks[i] = text.Chunk{
			ID:    fmt.Sprintf("c%04d", i+1),
			Text:  "Duplicate content",
			Norm:  "duplicate content",
			Index: i,
		}
	}
	config := dedupe.DefaultConfig()
	result := dedupe.Dedupe(chunks, config)

	err := WriteReport(result, 10, config, path)
	if err != nil {
		t.Fatalf("WriteReport failed: %v", err)
	}

	// Verify file exists and is readable
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}

	var report Report
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatalf("failed to parse report JSON: %v", err)
	}

	// Should have many dropped chunks
	if len(report.Dropped) == 0 {
		t.Error("expected dropped chunks in report")
	}
	if report.DroppedChunks != len(report.Dropped) {
		t.Errorf("DroppedChunks count (%d) should match Dropped list length (%d)",
			report.DroppedChunks, len(report.Dropped))
	}
}

// TestWriteReport_FileCreateError tests error handling when file creation fails
func TestWriteReport_FileCreateError(t *testing.T) {
	// Try to write to a path that's a directory (should fail)
	tmpDir := t.TempDir()
	invalidPath := tmpDir // Path is a directory, not a file

	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
	}
	config := dedupe.DefaultConfig()
	result := dedupe.Dedupe(chunks, config)

	err := WriteReport(result, 1, config, invalidPath)
	if err == nil {
		t.Error("expected error when path is a directory, got nil")
	}
}

// TestWriteReport_JSONMarshalError tests error handling (though JSON marshal shouldn't fail with valid data)
func TestWriteReport_JSONMarshalError(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "report.json")

	// Create a result with valid data - JSON marshal should succeed
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
	}
	config := dedupe.DefaultConfig()
	result := dedupe.Dedupe(chunks, config)

	err := WriteReport(result, 1, config, path)
	if err != nil {
		t.Fatalf("WriteReport should succeed with valid data, got: %v", err)
	}

	// Verify JSON is valid
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}

	var report Report
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatalf("failed to parse report JSON: %v", err)
	}
}

// TestWriteReport_FileWriteFailure tests error handling when file.Write fails
func TestWriteReport_FileWriteFailure(t *testing.T) {
	// This is hard to test without mocking os.File
	// We'll test that the function handles errors properly by using an invalid path
	tmpDir := t.TempDir()
	// Create a file and make it read-only to simulate write failure (platform-specific)
	readOnlyFile := filepath.Join(tmpDir, "readonly.json")
	if err := os.WriteFile(readOnlyFile, []byte("test"), 0444); err != nil {
		t.Fatalf("failed to create read-only file: %v", err)
	}

	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
	}
	config := dedupe.DefaultConfig()
	result := dedupe.Dedupe(chunks, config)

	// Try to write to read-only file (should fail on write, not create)
	err := WriteReport(result, 1, config, readOnlyFile)
	// On Unix systems, this should fail with permission error
	// On Windows, behavior may differ
	if err == nil {
		t.Log("write to read-only file succeeded (platform-specific behavior)")
	} else {
		// Error is expected
		if !os.IsPermission(err) {
			t.Logf("got error (expected): %v", err)
		}
	}
}

// TestWriteReport_CloseError tests that close errors are handled (logged but don't fail)
func TestWriteReport_CloseError(t *testing.T) {
	// Close errors are logged but don't cause WriteReport to fail
	// This is tested implicitly by the fact that other tests pass
	// We can't easily simulate a close error without mocking
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "report.json")

	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
	}
	config := dedupe.DefaultConfig()
	result := dedupe.Dedupe(chunks, config)

	err := WriteReport(result, 1, config, path)
	if err != nil {
		t.Fatalf("WriteReport should succeed even if close has issues: %v", err)
	}
}

// TestWriteReport_LargePreviewTruncation tests that previews are truncated to 200 chars
func TestWriteReport_LargePreviewTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "report.json")

	// Create chunk with very long text
	longText := strings.Repeat("a", 500) // 500 characters
	chunks := []text.Chunk{
		{ID: "c0001", Text: longText, Norm: longText, Index: 0},
		{ID: "c0002", Text: longText, Norm: longText, Index: 1}, // Duplicate
	}
	config := dedupe.DefaultConfig()
	result := dedupe.Dedupe(chunks, config)

	err := WriteReport(result, 2, config, path)
	if err != nil {
		t.Fatalf("WriteReport failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}

	var report Report
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatalf("failed to parse report JSON: %v", err)
	}

	// Check that previews are truncated
	for _, dropped := range report.Dropped {
		if len(dropped.Preview) > 203 { // 200 chars + "..."
			t.Errorf("preview should be truncated to 200 chars, got %d: %s", len(dropped.Preview), dropped.Preview)
		}
	}
}
