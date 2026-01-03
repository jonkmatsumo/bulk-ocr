package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListImages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []struct {
		name     string
		shouldInclude bool
	}{
		{"image1.jpg", true},
		{"image2.JPG", true},
		{"image3.jpeg", true},
		{"image4.JPEG", true},
		{"image5.png", true},
		{"image6.PNG", true},
		{"document.pdf", false},
		{"text.txt", false},
		{"image7.gif", false},
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", f.name, err)
		}
	}

	images, err := ListImages(tmpDir)
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}

	expectedCount := 6
	if len(images) != expectedCount {
		t.Errorf("expected %d images, got %d", expectedCount, len(images))
	}

	// Verify all returned files are images
	for _, img := range images {
		ext := filepath.Ext(img)
		ext = strings.ToLower(ext)
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			t.Errorf("unexpected extension in result: %s", img)
		}
	}
}

func TestListImages_NonExistentDirectory(t *testing.T) {
	_, err := ListImages("/nonexistent/directory")
	if err == nil {
		t.Error("expected error for non-existent directory, got nil")
	}
}

func TestListImages_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	images, err := ListImages(tmpDir)
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}

	if len(images) != 0 {
		t.Errorf("expected 0 images, got %d", len(images))
	}
}

func TestListImages_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	nestedDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}

	// Create files in root and nested
	files := []string{
		filepath.Join(tmpDir, "root.jpg"),
		filepath.Join(nestedDir, "nested.png"),
	}

	for _, f := range files {
		if err := os.WriteFile(f, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	images, err := ListImages(tmpDir)
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}

	if len(images) != 2 {
		t.Errorf("expected 2 images, got %d", len(images))
	}
}

func TestNaturalSort(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "simple lexicographic",
			input:    []string{"z.jpg", "a.jpg", "m.jpg"},
			expected: []string{"a.jpg", "m.jpg", "z.jpg"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single element",
			input:    []string{"image.jpg"},
			expected: []string{"image.jpg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NaturalSort(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d elements, got %d", len(tt.expected), len(result))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("position %d: expected %s, got %s", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

