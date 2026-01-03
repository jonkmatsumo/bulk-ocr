package text

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// Chunk represents a text chunk with original and normalized versions.
type Chunk struct {
	ID    string // sequential id: c0001, c0002, etc.
	Text  string // original text (trimmed, human-readable)
	Norm  string // normalized for hashing (lowercase, collapsed whitespace, no punctuation)
	Index int    // original position in document
}

// DefaultChromePatterns returns the default regex patterns for chrome filtering.
// Patterns are designed to match normalized text (lowercase, no punctuation).
func DefaultChromePatterns() []string {
	return []string{
		`\d{1,2}\s*\d{2}\s*(am|pm)?`,       // Timestamps (normalized: "1030 am" or "10 30 am")
		`\d+\s*%|wifi|battery|charging`,    // Battery/WiFi (normalized: lowercase)
		`back|forward|refresh|home|search`, // Browser chrome (normalized: lowercase)
		`\d{1,2}\s*\d{1,2}\s*\d{2,4}`,      // Date patterns (normalized: "1 1 2024")
	}
}

// Normalize normalizes text for hashing by lowercasing, collapsing whitespace, and removing punctuation.
// Preserves newlines for chunking boundaries.
func Normalize(raw string) string {
	if raw == "" {
		return ""
	}

	// Convert to lowercase
	normalized := strings.ToLower(raw)

	// Collapse multiple whitespace (but preserve newlines)
	// First, replace all non-newline whitespace with single space
	spaceRegex := regexp.MustCompile(`[ \t]+`)
	normalized = spaceRegex.ReplaceAllString(normalized, " ")

	// Collapse multiple newlines to single newline
	newlineRegex := regexp.MustCompile(`\n+`)
	normalized = newlineRegex.ReplaceAllString(normalized, "\n")

	// Remove punctuation (keep alphanumeric, spaces, and newlines)
	var result strings.Builder
	for _, r := range normalized {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '\n' {
			result.WriteRune(r)
		}
	}

	normalized = result.String()

	// Trim leading/trailing whitespace
	normalized = strings.TrimSpace(normalized)

	return normalized
}

// ChunkText splits text into chunks by paragraph boundaries (blank lines).
// Returns chunks with sequential IDs and normalized versions.
func ChunkText(text string, minChars int) []Chunk {
	if text == "" {
		return []Chunk{}
	}

	// Split on blank lines (one or more consecutive newlines)
	blankLineRegex := regexp.MustCompile(`\n\s*\n+`)
	segments := blankLineRegex.Split(text, -1)

	var chunks []Chunk
	chunkIndex := 0

	for _, segment := range segments {
		// Trim whitespace from segment
		trimmed := strings.TrimSpace(segment)

		// Skip if too short
		if len(trimmed) < minChars {
			continue
		}

		// Generate sequential ID
		chunkID := fmt.Sprintf("c%04d", chunkIndex+1)

		// Normalize for hashing
		normalized := Normalize(trimmed)

		chunk := Chunk{
			ID:    chunkID,
			Text:  trimmed,
			Norm:  normalized,
			Index: chunkIndex,
		}

		chunks = append(chunks, chunk)
		chunkIndex++
	}

	// If no blank lines found and text is long enough, create single chunk
	if len(chunks) == 0 && len(strings.TrimSpace(text)) >= minChars {
		trimmed := strings.TrimSpace(text)
		chunkID := fmt.Sprintf("c%04d", 1)
		normalized := Normalize(trimmed)
		chunks = append(chunks, Chunk{
			ID:    chunkID,
			Text:  trimmed,
			Norm:  normalized,
			Index: 0,
		})
	}

	return chunks
}

// FilterChrome removes chunks that match chrome patterns and are short.
// Only filters chunks that match pattern AND are below maxLength.
// Longer chunks matching patterns are kept (likely real content).
func FilterChrome(chunks []Chunk, patterns []string, maxLength int) []Chunk {
	if len(patterns) == 0 {
		return chunks
	}

	// Compile regex patterns
	compiledPatterns := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			// Skip invalid patterns (log would be better, but keeping simple)
			continue
		}
		compiledPatterns = append(compiledPatterns, re)
	}

	var filtered []Chunk

	for _, chunk := range chunks {
		shouldFilter := false

		// Check if chunk matches any pattern and is short
		if len(chunk.Norm) < maxLength {
			for _, re := range compiledPatterns {
				if re.MatchString(chunk.Norm) {
					shouldFilter = true
					break
				}
			}
		}

		if !shouldFilter {
			filtered = append(filtered, chunk)
		}
	}

	return filtered
}

// WriteChunksJSONL writes chunks to a JSONL file (one JSON object per line).
func WriteChunksJSONL(chunks []Chunk, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create JSONL file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			// Log the error, but don't return it as it's a cleanup error
			// The main error (if any) should have been returned already
			fmt.Fprintf(os.Stderr, "warning: failed to close JSONL file %s: %v\n", path, cerr)
		}
	}()

	writer := bufio.NewWriter(file)
	defer func() {
		if ferr := writer.Flush(); ferr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to flush JSONL writer for %s: %v\n", path, ferr)
		}
	}()

	for _, chunk := range chunks {
		// Truncate text to 500 chars for readability in JSON
		textPreview := chunk.Text
		if len(textPreview) > 500 {
			textPreview = textPreview[:500] + "..."
		}

		entry := map[string]interface{}{
			"id":    chunk.ID,
			"text":  textPreview,
			"index": chunk.Index,
			"len":   len(chunk.Text),
		}

		jsonData, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal chunk %s: %w", chunk.ID, err)
		}

		if _, err := writer.Write(jsonData); err != nil {
			return fmt.Errorf("failed to write chunk %s: %w", chunk.ID, err)
		}

		if _, err := writer.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	return nil
}

// RenderMarkdown renders chunks into Markdown format with a title.
// If includeChunkIDs is true, adds HTML comments before each chunk.
func RenderMarkdown(title string, chunks []Chunk, includeChunkIDs bool) string {
	// Use default title if empty
	if title == "" {
		title = "Extracted Notes"
	}

	var result strings.Builder
	// Write title header
	result.WriteString("# ")
	result.WriteString(title)
	result.WriteString("\n\n")

	// Write chunks
	for _, chunk := range chunks {
		if includeChunkIDs {
			// Add HTML comment with chunk ID
			result.WriteString("<!-- ")
			result.WriteString(chunk.ID)
			result.WriteString(" -->\n")
		}
		// Write chunk text
		result.WriteString(chunk.Text)
		// Add blank line separator
		result.WriteString("\n\n")
	}

	return result.String()
}

// WriteMarkdown writes Markdown content to a file with consistent line endings.
func WriteMarkdown(content string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create Markdown file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close Markdown file %s: %v\n", path, cerr)
		}
	}()

	writer := bufio.NewWriter(file)
	defer func() {
		if ferr := writer.Flush(); ferr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to flush Markdown writer for %s: %v\n", path, ferr)
		}
	}()

	// Normalize line endings to \n and ensure file ends with single newline
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	// Trim trailing newlines and add single newline
	normalized = strings.TrimRight(normalized, "\n")
	normalized += "\n"

	if _, err := writer.WriteString(normalized); err != nil {
		return fmt.Errorf("failed to write Markdown content: %w", err)
	}

	return nil
}
