package ingest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// ListImages walks a directory and returns all image file paths.
// Supported extensions: .jpg, .jpeg, .png (case-insensitive).
// If recursive is false, only scans the top-level directory.
// Returns absolute paths for reliable copying.
func ListImages(dir string, recursive bool) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}

	// Resolve to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	var images []string
	extensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// If not recursive and this is a subdirectory, skip it
			if !recursive && path != absDir {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if extensions[ext] {
			// Convert to absolute path
			absPath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("failed to resolve path: %w", err)
			}
			images = append(images, absPath)
		}
		return nil
	}

	err = filepath.Walk(absDir, walkFunc)
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return NaturalSort(images), nil
}

// NaturalSort sorts file paths using natural ordering.
// For example, "IMG_9.jpg" comes before "IMG_10.jpg".
func NaturalSort(paths []string) []string {
	sorted := make([]string, len(paths))
	copy(sorted, paths)

	sort.Slice(sorted, func(i, j int) bool {
		return naturalLess(sorted[i], sorted[j])
	})

	return sorted
}

// naturalLess compares two strings using natural ordering.
// For example, "IMG_9.jpg" comes before "IMG_10.jpg".
func naturalLess(a, b string) bool {
	// Extract base filenames for comparison
	baseA := filepath.Base(a)
	baseB := filepath.Base(b)

	// Split into segments (text and numbers)
	segmentsA := splitIntoSegments(baseA)
	segmentsB := splitIntoSegments(baseB)

	// Compare segment by segment
	maxLen := len(segmentsA)
	if len(segmentsB) > maxLen {
		maxLen = len(segmentsB)
	}

	for i := 0; i < maxLen; i++ {
		segA := ""
		segB := ""
		if i < len(segmentsA) {
			segA = segmentsA[i]
		}
		if i < len(segmentsB) {
			segB = segmentsB[i]
		}

		// If one segment is empty, the other is greater
		if segA == "" {
			return true
		}
		if segB == "" {
			return false
		}

		// Try to parse as numbers
		numA, errA := strconv.Atoi(segA)
		numB, errB := strconv.Atoi(segB)

		// Both are numbers: compare numerically
		if errA == nil && errB == nil {
			if numA != numB {
				return numA < numB
			}
			continue
		}

		// Both are text: compare lexicographically
		if errA != nil && errB != nil {
			if segA != segB {
				return segA < segB
			}
			continue
		}

		// One is number, one is text: numbers come before text
		if errA == nil {
			return true
		}
		return false
	}

	// If segments are equal, use full path as tie-breaker
	return a < b
}

// splitIntoSegments splits a string into alternating text and numeric segments.
// Example: "IMG_9.jpg" -> ["IMG_", "9", ".jpg"]
func splitIntoSegments(s string) []string {
	var segments []string
	var current strings.Builder
	var isDigit bool

	for _, r := range s {
		digit := unicode.IsDigit(r)
		if current.Len() == 0 {
			isDigit = digit
			current.WriteRune(r)
		} else if digit == isDigit {
			current.WriteRune(r)
		} else {
			segments = append(segments, current.String())
			current.Reset()
			isDigit = digit
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments
}

// StageImages copies images to a preprocessed directory with sequential names.
// Creates outDir/preprocessed/ and copies each image to 0001.jpg, 0002.png, etc.
// Preserves original extensions. Returns list of staged file paths (absolute).
func StageImages(imagePaths []string, outDir string) ([]string, error) {
	preprocessedDir := filepath.Join(outDir, "preprocessed")
	if err := os.MkdirAll(preprocessedDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create preprocessed directory: %w", err)
	}

	var stagedPaths []string

	for i, srcPath := range imagePaths {
		// Get original extension and normalize to lowercase
		ext := filepath.Ext(srcPath)
		if ext == "" {
			ext = ".jpg" // Default if no extension
		}
		ext = strings.ToLower(ext) // Normalize extension to lowercase

		// Generate sequential filename (zero-padded, 4 digits minimum)
		filename := fmt.Sprintf("%04d%s", i+1, ext)
		dstPath := filepath.Join(preprocessedDir, filename)

		// Copy file
		if err := copyFile(srcPath, dstPath); err != nil {
			return nil, fmt.Errorf("failed to copy %s to %s: %w", srcPath, dstPath, err)
		}

		// Resolve absolute path of destination
		absDstPath, err := filepath.Abs(dstPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve staged path: %w", err)
		}

		stagedPaths = append(stagedPaths, absDstPath)
	}

	return stagedPaths, nil
}

// copyFile copies a file from src to dst using io.Copy.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		if closeErr := srcFile.Close(); closeErr != nil {
			// Log but don't fail - file may already be closed
			_ = closeErr
		}
	}()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		if closeErr := dstFile.Close(); closeErr != nil {
			// Log but don't fail - file may already be closed
			_ = closeErr
		}
	}()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return dstFile.Close()
}
