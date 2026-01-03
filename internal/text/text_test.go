package text

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestNormalize_EmptyString(t *testing.T) {
	result := Normalize("")
	if result != "" {
		t.Errorf("expected empty string, got: %q", result)
	}
}

func TestNormalize_OnlyWhitespace(t *testing.T) {
	result := Normalize("   \n\n  \t  ")
	if result != "" {
		t.Errorf("expected empty string, got: %q", result)
	}
}

func TestNormalize_MixedCase(t *testing.T) {
	input := "Hello World!"
	expected := "hello world"
	result := Normalize(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestNormalize_Punctuation(t *testing.T) {
	input := "Hello, World! How are you?"
	expected := "hello world how are you"
	result := Normalize(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestNormalize_MultipleSpaces(t *testing.T) {
	input := "Hello    World"
	expected := "hello world"
	result := Normalize(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestNormalize_PreservesNewlines(t *testing.T) {
	input := "Hello\n\nWorld"
	result := Normalize(input)
	// Should preserve newline structure for chunking
	if !strings.Contains(result, "\n") {
		t.Errorf("expected newline to be preserved, got: %q", result)
	}
}

func TestNormalize_Unicode(t *testing.T) {
	input := "Café, naïve, résumé"
	result := Normalize(input)
	// Should handle unicode characters
	if result == "" {
		t.Errorf("expected non-empty result for unicode input, got: %q", result)
	}
}

func TestNormalize_OnlyNumbers(t *testing.T) {
	input := "123 456 789"
	expected := "123 456 789"
	result := Normalize(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestChunk_EmptyInput(t *testing.T) {
	result := ChunkText("", 60)
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d chunks", len(result))
	}
}

func TestChunk_SingleParagraph(t *testing.T) {
	text := "This is a single paragraph with enough text to pass the minimum character threshold for chunking."
	result := ChunkText(text, 60)
	if len(result) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(result))
	}
	if result[0].ID != "c0001" {
		t.Errorf("expected ID c0001, got %s", result[0].ID)
	}
	if result[0].Index != 0 {
		t.Errorf("expected index 0, got %d", result[0].Index)
	}
}

func TestChunk_MultipleParagraphs(t *testing.T) {
	text := "First paragraph with enough text to pass the minimum character threshold.\n\nSecond paragraph with enough text to pass the minimum character threshold.\n\nThird paragraph with enough text to pass the minimum character threshold."
	result := ChunkText(text, 60)
	if len(result) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(result))
	}
	if result[0].ID != "c0001" {
		t.Errorf("expected ID c0001, got %s", result[0].ID)
	}
	if result[1].ID != "c0002" {
		t.Errorf("expected ID c0002, got %s", result[1].ID)
	}
	if result[2].ID != "c0003" {
		t.Errorf("expected ID c0003, got %s", result[2].ID)
	}
}

func TestChunk_ChunksBelowThreshold(t *testing.T) {
	text := "Short.\n\nAlso short.\n\nThis is a longer paragraph that should pass the minimum character threshold and be included in the chunks."
	result := ChunkText(text, 60)
	// Should only include the long paragraph
	if len(result) != 1 {
		t.Errorf("expected 1 chunk (only long paragraph), got %d", len(result))
	}
}

func TestChunk_AllChunksTooShort(t *testing.T) {
	text := "Short.\n\nAlso short."
	result := ChunkText(text, 60)
	if len(result) != 0 {
		t.Errorf("expected 0 chunks, got %d", len(result))
	}
}

func TestChunk_MixedLineEndings(t *testing.T) {
	text := "First paragraph with enough text to pass the minimum character threshold.\r\n\r\nSecond paragraph with enough text to pass the minimum character threshold.\n\nThird paragraph with enough text to pass the minimum character threshold."
	result := ChunkText(text, 60)
	// Should handle both \n and \r\n
	if len(result) < 2 {
		t.Errorf("expected at least 2 chunks, got %d", len(result))
	}
}

func TestChunk_ConsecutiveBlankLines(t *testing.T) {
	text := "First paragraph with enough text to pass the minimum character threshold.\n\n\n\nSecond paragraph with enough text to pass the minimum character threshold."
	result := ChunkText(text, 60)
	// Should split on multiple blank lines
	if len(result) < 2 {
		t.Errorf("expected at least 2 chunks, got %d", len(result))
	}
}

func TestChunk_SequentialIDs(t *testing.T) {
	text := "First paragraph with enough text to pass the minimum character threshold for chunking.\n\nSecond paragraph with enough text to pass the minimum character threshold for chunking.\n\nThird paragraph with enough text to pass the minimum character threshold for chunking."
	result := ChunkText(text, 60)
	expectedIDs := []string{"c0001", "c0002", "c0003"}
	actualIDs := make([]string, len(result))
	for i, chunk := range result {
		actualIDs[i] = chunk.ID
	}
	if !reflect.DeepEqual(actualIDs, expectedIDs) {
		t.Errorf("expected IDs %v, got %v", expectedIDs, actualIDs)
	}
}

func TestChunk_NormalizationPreserved(t *testing.T) {
	text := "Hello, World! This is a test with enough text to pass the minimum character threshold for chunking."
	result := ChunkText(text, 60)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	// Original text should be preserved
	if result[0].Text != strings.TrimSpace(text) {
		t.Errorf("original text not preserved: got %q", result[0].Text)
	}
	// Normalized version should be different
	if result[0].Norm == result[0].Text {
		t.Errorf("normalized text should differ from original")
	}
	// Normalized should be lowercase and without punctuation
	if strings.Contains(result[0].Norm, ",") || strings.Contains(result[0].Norm, "!") {
		t.Errorf("normalized text should not contain punctuation: %q", result[0].Norm)
	}
}

func TestFilterChrome_NoPatterns(t *testing.T) {
	chunks := []Chunk{
		{ID: "c0001", Text: "Test chunk", Norm: "test chunk", Index: 0},
	}
	result := FilterChrome(chunks, []string{}, 100)
	if len(result) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(result))
	}
}

func TestFilterChrome_TimestampPattern(t *testing.T) {
	chunks := []Chunk{
		{ID: "c0001", Text: "10:30 AM", Norm: "1030 am", Index: 0}, // Short, matches pattern
		{ID: "c0002", Text: "This is a longer chunk that contains 10:30 AM but should be kept because it's long enough.", Norm: "this is a longer chunk that contains 1030 am but should be kept because its long enough", Index: 1}, // Long, matches pattern but should keep
		{ID: "c0003", Text: "Regular content here", Norm: "regular content here", Index: 2}, // Doesn't match
	}
	patterns := []string{`\d{1,2}\s*\d{2}\s*(am|pm)?`} // Pattern for normalized text
	result := FilterChrome(chunks, patterns, 50)
	// Should filter c0001 (short + matches), keep c0002 (long + matches), keep c0003 (doesn't match)
	if len(result) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(result))
	}
	if result[0].ID != "c0002" {
		t.Errorf("expected c0002 to be first, got %s", result[0].ID)
	}
	if result[1].ID != "c0003" {
		t.Errorf("expected c0003 to be second, got %s", result[1].ID)
	}
}

func TestFilterChrome_BatteryPattern(t *testing.T) {
	chunks := []Chunk{
		{ID: "c0001", Text: "85%", Norm: "85", Index: 0},                          // Short, but "85" doesn't match pattern (needs %)
		{ID: "c0002", Text: "Battery", Norm: "battery", Index: 1},                 // Short, matches
		{ID: "c0003", Text: "Regular content", Norm: "regular content", Index: 2}, // Doesn't match
	}
	patterns := []string{`\d+\s*%|wifi|battery|charging`} // Pattern for normalized text
	result := FilterChrome(chunks, patterns, 50)
	// Should filter c0002 (short + matches), keep c0001 (doesn't match pattern), keep c0003 (doesn't match)
	// Note: "85" normalized doesn't have %, so it won't match the pattern
	if len(result) != 2 {
		t.Errorf("expected 2 chunks (c0001 and c0003), got %d", len(result))
	}
	// c0001 should be kept (doesn't match pattern), c0003 should be kept
	foundC0003 := false
	for _, chunk := range result {
		if chunk.ID == "c0003" {
			foundC0003 = true
			break
		}
	}
	if !foundC0003 {
		t.Errorf("expected c0003 to be kept")
	}
}

func TestFilterChrome_MultiplePatterns(t *testing.T) {
	chunks := []Chunk{
		{ID: "c0001", Text: "10:30", Norm: "1030", Index: 0},                      // Matches timestamp pattern
		{ID: "c0002", Text: "Back", Norm: "back", Index: 1},                       // Matches browser chrome
		{ID: "c0003", Text: "Regular content", Norm: "regular content", Index: 2}, // Doesn't match
	}
	patterns := DefaultChromePatterns()
	result := FilterChrome(chunks, patterns, 50)
	// Should filter c0001 and c0002, keep c0003
	if len(result) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(result))
	}
	if result[0].ID != "c0003" {
		t.Errorf("expected c0003, got %s", result[0].ID)
	}
}

func TestFilterChrome_InvalidPattern(t *testing.T) {
	chunks := []Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
	}
	patterns := []string{`[invalid regex(`} // Invalid regex
	result := FilterChrome(chunks, patterns, 50)
	// Should keep all chunks if pattern is invalid
	if len(result) != 1 {
		t.Errorf("expected 1 chunk (invalid pattern ignored), got %d", len(result))
	}
}

func TestWriteChunksJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "chunks.jsonl")

	chunks := []Chunk{
		{ID: "c0001", Text: "First chunk", Norm: "first chunk", Index: 0},
		{ID: "c0002", Text: "Second chunk", Norm: "second chunk", Index: 1},
	}

	err := WriteChunksJSONL(chunks, path)
	if err != nil {
		t.Fatalf("WriteChunksJSONL failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("JSONL file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read JSONL file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}

	// Verify first line contains expected data
	if !strings.Contains(lines[0], `"id":"c0001"`) {
		t.Errorf("expected c0001 in first line, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], `"text":"First chunk"`) {
		t.Errorf("expected text in first line, got: %s", lines[0])
	}
}

func TestWriteChunksJSONL_LongTextTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "chunks.jsonl")

	longText := strings.Repeat("a", 600)
	chunks := []Chunk{
		{ID: "c0001", Text: longText, Norm: longText, Index: 0},
	}

	err := WriteChunksJSONL(chunks, path)
	if err != nil {
		t.Fatalf("WriteChunksJSONL failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read JSONL file: %v", err)
	}

	// Verify text is truncated
	if !strings.Contains(string(content), "...") {
		t.Error("expected text to be truncated with ...")
	}
	if strings.Contains(string(content), longText) {
		t.Error("expected long text to be truncated")
	}
}

func TestDefaultChromePatterns(t *testing.T) {
	patterns := DefaultChromePatterns()
	if len(patterns) == 0 {
		t.Error("expected default patterns, got empty slice")
	}
	// Verify patterns are valid regex
	for _, pattern := range patterns {
		_, err := regexp.Compile(pattern)
		if err != nil {
			t.Errorf("invalid regex pattern %q: %v", pattern, err)
		}
	}
}

func TestChunk_EdgeCase_OnlyNewlines(t *testing.T) {
	text := "\n\n\n"
	result := ChunkText(text, 60)
	if len(result) != 0 {
		t.Errorf("expected 0 chunks for only newlines, got %d", len(result))
	}
}

func TestChunk_EdgeCase_NoNewlines(t *testing.T) {
	text := "This is a single long paragraph with no newlines that should still be chunked if it meets the minimum character requirement."
	result := ChunkText(text, 60)
	if len(result) != 1 {
		t.Errorf("expected 1 chunk for single paragraph, got %d", len(result))
	}
}

func TestChunk_EdgeCase_VeryLongParagraph(t *testing.T) {
	// Create a very long paragraph (should still be chunked as one)
	longText := strings.Repeat("This is a sentence. ", 100)
	result := ChunkText(longText, 60)
	if len(result) != 1 {
		t.Errorf("expected 1 chunk for very long paragraph, got %d", len(result))
	}
}

func TestNormalize_EdgeCase_OnlyPunctuation(t *testing.T) {
	result := Normalize("!!!???")
	if result != "" {
		t.Errorf("expected empty string for only punctuation, got %q", result)
	}
}

func TestNormalize_EdgeCase_MixedWhitespace(t *testing.T) {
	input := "Hello\t\tWorld\n\nTest"
	result := Normalize(input)
	// Should collapse tabs and spaces, preserve newlines
	if strings.Contains(result, "\t") {
		t.Errorf("expected tabs to be collapsed, got: %q", result)
	}
}
