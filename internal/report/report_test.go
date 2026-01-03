package report

import (
	"encoding/json"
	"os"
	"path/filepath"
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
}
