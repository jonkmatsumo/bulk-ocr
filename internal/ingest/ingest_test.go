package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListImages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []struct {
		name          string
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

	images, err := ListImages(tmpDir, true)
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
	_, err := ListImages("/nonexistent/directory", true)
	if err == nil {
		t.Error("expected error for non-existent directory, got nil")
	}
}

func TestListImages_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	images, err := ListImages(tmpDir, true)
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

	images, err := ListImages(tmpDir, true)
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}

	if len(images) != 2 {
		t.Errorf("expected 2 images, got %d", len(images))
	}
}

func TestListImages_Recursive(t *testing.T) {
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

	// Test recursive=true
	images, err := ListImages(tmpDir, true)
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}
	if len(images) != 2 {
		t.Errorf("expected 2 images with recursive=true, got %d", len(images))
	}

	// Test recursive=false
	images, err = ListImages(tmpDir, false)
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}
	if len(images) != 1 {
		t.Errorf("expected 1 image with recursive=false, got %d", len(images))
	}
	if !strings.Contains(images[0], "root.jpg") {
		t.Errorf("expected root.jpg with recursive=false, got %s", images[0])
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
		{
			name:     "natural sort numbers",
			input:    []string{"IMG_10.jpg", "IMG_9.jpg", "IMG_2.jpg"},
			expected: []string{"IMG_2.jpg", "IMG_9.jpg", "IMG_10.jpg"},
		},
		{
			name:     "natural sort with paths",
			input:    []string{"/path/IMG_10.jpg", "/path/IMG_9.jpg"},
			expected: []string{"/path/IMG_9.jpg", "/path/IMG_10.jpg"},
		},
		{
			name:     "mixed alphanumeric",
			input:    []string{"image_001.png", "image_002.png", "image_010.png"},
			expected: []string{"image_001.png", "image_002.png", "image_010.png"},
		},
		{
			name:     "numbers before text",
			input:    []string{"10test.jpg", "2test.jpg", "test.jpg"},
			expected: []string{"2test.jpg", "10test.jpg", "test.jpg"},
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

func TestNaturalSort_Stability(t *testing.T) {
	// Test that sorting is stable across multiple runs
	input := []string{
		"/path/IMG_10.jpg",
		"/path/IMG_9.jpg",
		"/path/IMG_2.jpg",
		"/other/IMG_1.jpg",
	}

	// Sort multiple times
	result1 := NaturalSort(input)
	result2 := NaturalSort(input)
	result3 := NaturalSort(input)

	// All should be identical
	if len(result1) != len(result2) || len(result2) != len(result3) {
		t.Error("sort results have different lengths")
	}

	for i := range result1 {
		if result1[i] != result2[i] || result2[i] != result3[i] {
			t.Errorf("sort is not stable: position %d differs", i)
		}
	}
}

func TestStageImages(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := t.TempDir()

	// Create test images
	testFiles := []string{"test1.jpg", "test2.png", "test3.jpeg"}
	var imagePaths []string
	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test image data"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		absPath, _ := filepath.Abs(path)
		imagePaths = append(imagePaths, absPath)
	}

	// Stage images
	staged, err := StageImages(imagePaths, outDir)
	if err != nil {
		t.Fatalf("StageImages failed: %v", err)
	}

	if len(staged) != len(testFiles) {
		t.Errorf("expected %d staged files, got %d", len(testFiles), len(staged))
	}

	// Verify files exist with correct names
	expectedNames := []string{"0001.jpg", "0002.png", "0003.jpeg"}
	preprocessedDir := filepath.Join(outDir, "preprocessed")
	for i, expectedName := range expectedNames {
		expectedPath := filepath.Join(preprocessedDir, expectedName)
		if staged[i] != expectedPath {
			// Check if absolute paths match
			absExpected, _ := filepath.Abs(expectedPath)
			if staged[i] != absExpected {
				t.Errorf("position %d: expected %s, got %s", i, absExpected, staged[i])
			}
		}

		// Verify file exists
		if _, err := os.Stat(staged[i]); os.IsNotExist(err) {
			t.Errorf("staged file does not exist: %s", staged[i])
		}

		// Verify file content
		content, err := os.ReadFile(staged[i])
		if err != nil {
			t.Errorf("failed to read staged file: %v", err)
		}
		if string(content) != "test image data" {
			t.Errorf("staged file content incorrect")
		}
	}
}

func TestStageImages_EmptyInput(t *testing.T) {
	outDir := t.TempDir()

	staged, err := StageImages([]string{}, outDir)
	if err != nil {
		t.Fatalf("StageImages failed: %v", err)
	}

	if len(staged) != 0 {
		t.Errorf("expected 0 staged files, got %d", len(staged))
	}

	// Verify preprocessed directory was created
	preprocessedDir := filepath.Join(outDir, "preprocessed")
	if _, err := os.Stat(preprocessedDir); os.IsNotExist(err) {
		t.Error("preprocessed directory was not created")
	}
}

func TestStageImages_PreservesExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := t.TempDir()

	// Create images with different extensions
	extensions := []string{".jpg", ".png", ".jpeg", ".JPG", ".PNG"}
	var imagePaths []string
	for i, ext := range extensions {
		filename := fmt.Sprintf("test%d%s", i, ext)
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		absPath, _ := filepath.Abs(path)
		imagePaths = append(imagePaths, absPath)
	}

	staged, err := StageImages(imagePaths, outDir)
	if err != nil {
		t.Fatalf("StageImages failed: %v", err)
	}

	// Verify extensions are preserved (case normalized)
	for i, stagedPath := range staged {
		ext := filepath.Ext(stagedPath)
		expectedExt := strings.ToLower(extensions[i])
		if ext != expectedExt {
			t.Errorf("position %d: expected extension %s, got %s", i, expectedExt, ext)
		}

		// Verify filename format
		filename := filepath.Base(stagedPath)
		expectedFilename := fmt.Sprintf("%04d%s", i+1, expectedExt)
		if filename != expectedFilename {
			t.Errorf("position %d: expected filename %s, got %s", i, expectedFilename, filename)
		}
	}
}

func TestStageImages_SequentialNaming(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := t.TempDir()

	// Create many images to test zero-padding
	numImages := 150
	var imagePaths []string
	for i := 0; i < numImages; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("img%d.jpg", i))
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		absPath, _ := filepath.Abs(path)
		imagePaths = append(imagePaths, absPath)
	}

	staged, err := StageImages(imagePaths, outDir)
	if err != nil {
		t.Fatalf("StageImages failed: %v", err)
	}

	if len(staged) != numImages {
		t.Errorf("expected %d staged files, got %d", numImages, len(staged))
	}

	// Verify naming: 0001.jpg, 0002.jpg, ..., 0150.jpg
	preprocessedDir := filepath.Join(outDir, "preprocessed")
	for i := 0; i < numImages; i++ {
		expectedName := fmt.Sprintf("%04d.jpg", i+1)
		expectedPath := filepath.Join(preprocessedDir, expectedName)
		absExpected, _ := filepath.Abs(expectedPath)

		if staged[i] != absExpected {
			t.Errorf("position %d: expected %s, got %s", i, absExpected, staged[i])
		}
	}
}
