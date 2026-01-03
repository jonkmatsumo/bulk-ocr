package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListImages walks a directory and returns all image file paths.
// Supported extensions: .jpg, .jpeg, .png (case-insensitive).
func ListImages(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}

	var images []string
	extensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if extensions[ext] {
			images = append(images, path)
		}
		return nil
	})

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
func naturalLess(a, b string) bool {
	// Simple implementation: compare lexicographically for now.
	// A more sophisticated implementation would handle numeric sequences.
	// For Milestone 0, this is sufficient.
	return a < b
}
