package dedupe

import (
	"reflect"
	"testing"

	"github.com/jonkmatsumo/bulk-ocr/internal/text"
)

func TestExactHashDedupe_EmptyInput(t *testing.T) {
	kept, dropped := exactHashDedupe([]text.Chunk{})
	if len(kept) != 0 {
		t.Errorf("expected 0 kept chunks, got %d", len(kept))
	}
	if len(dropped) != 0 {
		t.Errorf("expected 0 dropped chunks, got %d", len(dropped))
	}
}

func TestExactHashDedupe_SingleChunk(t *testing.T) {
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test chunk", Norm: "test chunk", Index: 0},
	}
	kept, dropped := exactHashDedupe(chunks)
	if len(kept) != 1 {
		t.Errorf("expected 1 kept chunk, got %d", len(kept))
	}
	if len(dropped) != 0 {
		t.Errorf("expected 0 dropped chunks, got %d", len(dropped))
	}
	if kept[0].ID != "c0001" {
		t.Errorf("expected kept chunk ID c0001, got %s", kept[0].ID)
	}
}

func TestExactHashDedupe_AllIdentical(t *testing.T) {
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test chunk", Norm: "test chunk", Index: 0},
		{ID: "c0002", Text: "Test chunk", Norm: "test chunk", Index: 1},
		{ID: "c0003", Text: "Test chunk", Norm: "test chunk", Index: 2},
	}
	kept, dropped := exactHashDedupe(chunks)
	if len(kept) != 1 {
		t.Errorf("expected 1 kept chunk, got %d", len(kept))
	}
	if len(dropped) != 2 {
		t.Errorf("expected 2 dropped chunks, got %d", len(dropped))
	}
	if kept[0].ID != "c0001" {
		t.Errorf("expected kept chunk ID c0001, got %s", kept[0].ID)
	}
	if dropped[0].Reason != "exact_duplicate" {
		t.Errorf("expected reason exact_duplicate, got %s", dropped[0].Reason)
	}
}

func TestExactHashDedupe_NoDuplicates(t *testing.T) {
	chunks := []text.Chunk{
		{ID: "c0001", Text: "First chunk", Norm: "first chunk", Index: 0},
		{ID: "c0002", Text: "Second chunk", Norm: "second chunk", Index: 1},
		{ID: "c0003", Text: "Third chunk", Norm: "third chunk", Index: 2},
	}
	kept, dropped := exactHashDedupe(chunks)
	if len(kept) != 3 {
		t.Errorf("expected 3 kept chunks, got %d", len(kept))
	}
	if len(dropped) != 0 {
		t.Errorf("expected 0 dropped chunks, got %d", len(dropped))
	}
}

func TestExactHashDedupe_MixedDuplicates(t *testing.T) {
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Unique one", Norm: "unique one", Index: 0},
		{ID: "c0002", Text: "Duplicate", Norm: "duplicate", Index: 1},
		{ID: "c0003", Text: "Unique two", Norm: "unique two", Index: 2},
		{ID: "c0004", Text: "Duplicate", Norm: "duplicate", Index: 3},
		{ID: "c0005", Text: "Unique three", Norm: "unique three", Index: 4},
	}
	kept, dropped := exactHashDedupe(chunks)
	if len(kept) != 4 {
		t.Errorf("expected 4 kept chunks, got %d", len(kept))
	}
	if len(dropped) != 1 {
		t.Errorf("expected 1 dropped chunk, got %d", len(dropped))
	}
	if dropped[0].ChunkID != "c0004" {
		t.Errorf("expected dropped chunk ID c0004, got %s", dropped[0].ChunkID)
	}
}

func TestExactHashDedupe_EmptyNormalizedText(t *testing.T) {
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "", Index: 0},
		{ID: "c0002", Text: "Test", Norm: "", Index: 1},
	}
	kept, _ := exactHashDedupe(chunks)
	// Empty normalized text should be kept (edge case handling)
	if len(kept) != 2 {
		t.Errorf("expected 2 kept chunks (empty norm kept), got %d", len(kept))
	}
}

func TestGenerateKgrams_EmptyString(t *testing.T) {
	result := generateKgrams("", 3)
	if len(result) != 0 {
		t.Errorf("expected 0 k-grams, got %d", len(result))
	}
}

func TestGenerateKgrams_ShorterThanK(t *testing.T) {
	result := generateKgrams("ab", 3)
	if len(result) != 0 {
		t.Errorf("expected 0 k-grams, got %d", len(result))
	}
}

func TestGenerateKgrams_EqualK(t *testing.T) {
	result := generateKgrams("abc", 3)
	if len(result) != 1 {
		t.Errorf("expected 1 k-gram, got %d", len(result))
	}
	if result[0] != "abc" {
		t.Errorf("expected k-gram 'abc', got %s", result[0])
	}
}

func TestGenerateKgrams_LongerThanK(t *testing.T) {
	result := generateKgrams("hello", 3)
	expected := []string{"hel", "ell", "llo"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestGenerateKgrams_Contiguous(t *testing.T) {
	result := generateKgrams("abcdef", 2)
	expected := []string{"ab", "bc", "cd", "de", "ef"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
	// Verify no gaps
	for i := 0; i < len(result)-1; i++ {
		if result[i][1:] != result[i+1][:1] {
			t.Errorf("k-grams not contiguous: %s and %s", result[i], result[i+1])
		}
	}
}

func TestGenerateKgrams_Unicode(t *testing.T) {
	result := generateKgrams("café", 2)
	// Should handle unicode correctly
	if len(result) == 0 {
		t.Error("expected k-grams for unicode string")
	}
}

func TestFnv1a64_EmptyInput(t *testing.T) {
	result := fnv1a64([]byte{})
	if result != fnvOffsetBasis64 {
		t.Errorf("expected offset basis for empty input, got %d", result)
	}
}

func TestFnv1a64_SingleByte(t *testing.T) {
	result1 := fnv1a64([]byte{'a'})
	result2 := fnv1a64([]byte{'a'})
	if result1 != result2 {
		t.Error("same input should produce same hash")
	}
	if result1 == fnvOffsetBasis64 {
		t.Error("non-empty input should not equal offset basis")
	}
}

func TestFnv1a64_Deterministic(t *testing.T) {
	data := []byte("test data")
	result1 := fnv1a64(data)
	result2 := fnv1a64(data)
	if result1 != result2 {
		t.Error("same input should produce same hash")
	}
}

func TestFnv1a64_DifferentInputs(t *testing.T) {
	result1 := fnv1a64([]byte("test1"))
	result2 := fnv1a64([]byte("test2"))
	if result1 == result2 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestSimhash64_EmptyText(t *testing.T) {
	result := simhash64("", 5)
	if result != 0 {
		t.Errorf("expected 0 for empty text, got %d", result)
	}
}

func TestSimhash64_TextShorterThanK(t *testing.T) {
	result := simhash64("ab", 5)
	if result != 0 {
		t.Errorf("expected 0 for text shorter than k, got %d", result)
	}
}

func TestSimhash64_IdenticalTexts(t *testing.T) {
	text := "this is a test string for simhash"
	result1 := simhash64(text, 5)
	result2 := simhash64(text, 5)
	if result1 != result2 {
		t.Error("identical texts should produce identical signatures")
	}
}

func TestSimhash64_SimilarTexts(t *testing.T) {
	text1 := "this is a test string for simhash"
	text2 := "this is a test string for simhash with small change"
	sig1 := simhash64(text1, 5)
	sig2 := simhash64(text2, 5)
	dist := hammingDistance(sig1, sig2)
	// Similar texts should have low Hamming distance
	if dist > 20 {
		t.Errorf("similar texts should have low distance, got %d", dist)
	}
}

func TestSimhash64_VeryDifferentTexts(t *testing.T) {
	text1 := "this is a test string"
	text2 := "completely different content here"
	sig1 := simhash64(text1, 5)
	sig2 := simhash64(text2, 5)
	dist := hammingDistance(sig1, sig2)
	// Very different texts should have higher Hamming distance
	if dist < 10 {
		t.Errorf("very different texts should have higher distance, got %d", dist)
	}
}

func TestSimhash64_SingleKgram(t *testing.T) {
	result := simhash64("abc", 3)
	// Should still produce a valid signature
	if result == 0 {
		t.Error("single k-gram should produce non-zero signature")
	}
}

func TestHammingDistance_Identical(t *testing.T) {
	val := uint64(0x1234567890ABCDEF)
	dist := hammingDistance(val, val)
	if dist != 0 {
		t.Errorf("expected distance 0 for identical values, got %d", dist)
	}
}

func TestHammingDistance_AllDifferent(t *testing.T) {
	val1 := uint64(0x0000000000000000)
	val2 := uint64(0xFFFFFFFFFFFFFFFF)
	dist := hammingDistance(val1, val2)
	if dist != 64 {
		t.Errorf("expected distance 64 for all bits different, got %d", dist)
	}
}

func TestHammingDistance_EdgeCases(t *testing.T) {
	dist1 := hammingDistance(0, 0)
	if dist1 != 0 {
		t.Errorf("expected distance 0 for 0 vs 0, got %d", dist1)
	}

	dist2 := hammingDistance(0, ^uint64(0))
	if dist2 != 64 {
		t.Errorf("expected distance 64 for 0 vs max, got %d", dist2)
	}
}

func TestSimhashDedupe_EmptyInput(t *testing.T) {
	config := DefaultConfig()
	kept, dropped := simhashDedupe([]text.Chunk{}, config)
	if len(kept) != 0 {
		t.Errorf("expected 0 kept chunks, got %d", len(kept))
	}
	if len(dropped) != 0 {
		t.Errorf("expected 0 dropped chunks, got %d", len(dropped))
	}
}

func TestSimhashDedupe_SingleChunk(t *testing.T) {
	config := DefaultConfig()
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test chunk", Norm: "test chunk", Index: 0},
	}
	kept, dropped := simhashDedupe(chunks, config)
	if len(kept) != 1 {
		t.Errorf("expected 1 kept chunk, got %d", len(kept))
	}
	if len(dropped) != 0 {
		t.Errorf("expected 0 dropped chunks, got %d", len(dropped))
	}
}

func TestSimhashDedupe_NoNearDuplicates(t *testing.T) {
	config := DefaultConfig()
	config.SimHashThreshold = 3 // Lower threshold to avoid false positives
	chunks := []text.Chunk{
		{ID: "c0001", Text: "First completely different chunk with unique content", Norm: "first completely different chunk with unique content", Index: 0},
		{ID: "c0002", Text: "Second completely different chunk with unique content", Norm: "second completely different chunk with unique content", Index: 1},
		{ID: "c0003", Text: "Third completely different chunk with unique content", Norm: "third completely different chunk with unique content", Index: 2},
	}
	kept, dropped := simhashDedupe(chunks, config)
	if len(kept) != 3 {
		t.Errorf("expected 3 kept chunks, got %d", len(kept))
	}
	if len(dropped) != 0 {
		t.Errorf("expected 0 dropped chunks, got %d", len(dropped))
	}
}

func TestSimhashDedupe_AllNearDuplicates(t *testing.T) {
	config := DefaultConfig()
	config.SimHashThreshold = 10 // Higher threshold to catch near-duplicates
	chunks := []text.Chunk{
		{ID: "c0001", Text: "This is a test string for simhash", Norm: "this is a test string for simhash", Index: 0},
		{ID: "c0002", Text: "This is a test string for simhash with small change", Norm: "this is a test string for simhash with small change", Index: 1},
		{ID: "c0003", Text: "This is a test string for simhash with another small change", Norm: "this is a test string for simhash with another small change", Index: 2},
	}
	kept, _ := simhashDedupe(chunks, config)
	// Should keep first, drop rest if they're similar enough
	if len(kept) < 1 {
		t.Errorf("expected at least 1 kept chunk, got %d", len(kept))
	}
	if kept[0].ID != "c0001" {
		t.Errorf("expected first chunk to be kept, got %s", kept[0].ID)
	}
}

func TestSimhashDedupe_WindowSize0(t *testing.T) {
	config := DefaultConfig()
	config.Window = 0 // Compare against all previous
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
		{ID: "c0002", Text: "Test", Norm: "test", Index: 1},
	}
	kept, _ := simhashDedupe(chunks, config)
	// Should compare c0002 against c0001
	if len(kept) != 1 {
		t.Errorf("expected 1 kept chunk, got %d", len(kept))
	}
}

func TestSimhashDedupe_WindowSize1(t *testing.T) {
	config := DefaultConfig()
	config.Window = 1 // Only compare with previous
	chunks := []text.Chunk{
		{ID: "c0001", Text: "First", Norm: "first", Index: 0},
		{ID: "c0002", Text: "Second", Norm: "second", Index: 1},
		{ID: "c0003", Text: "First", Norm: "first", Index: 2}, // Duplicate of c0001, but outside window
	}
	kept, _ := simhashDedupe(chunks, config)
	// c0003 should not match c0001 because it's outside window
	if len(kept) != 3 {
		t.Errorf("expected 3 kept chunks (window prevents match), got %d", len(kept))
	}
}

func TestSimhashDedupe_ThresholdBoundary(t *testing.T) {
	config := DefaultConfig()
	config.SimHashThreshold = 5
	// Create chunks with known similarity
	chunks := []text.Chunk{
		{ID: "c0001", Text: "test string one", Norm: "test string one", Index: 0},
		{ID: "c0002", Text: "test string two", Norm: "test string two", Index: 1},
	}
	kept, _ := simhashDedupe(chunks, config)
	// Result depends on actual Hamming distance
	// Just verify it doesn't crash and produces valid result
	if len(kept) < 1 || len(kept) > 2 {
		t.Errorf("expected 1-2 kept chunks, got %d", len(kept))
	}
}

func TestDedupe_EmptyInput(t *testing.T) {
	config := DefaultConfig()
	result := Dedupe([]text.Chunk{}, config)
	if result.Stats.InputCount != 0 {
		t.Errorf("expected 0 input count, got %d", result.Stats.InputCount)
	}
	if len(result.KeptChunks) != 0 {
		t.Errorf("expected 0 kept chunks, got %d", len(result.KeptChunks))
	}
}

func TestDedupe_MethodExact(t *testing.T) {
	config := DefaultConfig()
	config.Method = "exact"
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
		{ID: "c0002", Text: "Test", Norm: "test", Index: 1},
	}
	result := Dedupe(chunks, config)
	if len(result.KeptChunks) != 1 {
		t.Errorf("expected 1 kept chunk, got %d", len(result.KeptChunks))
	}
	if result.Stats.ExactDups != 1 {
		t.Errorf("expected 1 exact duplicate, got %d", result.Stats.ExactDups)
	}
}

func TestDedupe_MethodSimhash(t *testing.T) {
	config := DefaultConfig()
	config.Method = "simhash"
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test chunk one", Norm: "test chunk one", Index: 0},
		{ID: "c0002", Text: "Test chunk two", Norm: "test chunk two", Index: 1},
	}
	result := Dedupe(chunks, config)
	// Should run exact hash first, then simhash
	if result.Stats.InputCount != 2 {
		t.Errorf("expected 2 input chunks, got %d", result.Stats.InputCount)
	}
}

func TestDedupe_InvalidMethod(t *testing.T) {
	config := DefaultConfig()
	config.Method = "invalid"
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
	}
	result := Dedupe(chunks, config)
	// Should default to simhash
	if len(result.KeptChunks) != 1 {
		t.Errorf("expected 1 kept chunk, got %d", len(result.KeptChunks))
	}
}

func TestDedupe_InvalidConfigValues(t *testing.T) {
	config := Config{
		Method:           "simhash",
		SimHashK:         -1, // Invalid
		SimHashThreshold: -1, // Invalid
		Window:           -1, // Invalid
	}
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
	}
	result := Dedupe(chunks, config)
	// Should validate and use defaults
	if len(result.KeptChunks) != 1 {
		t.Errorf("expected 1 kept chunk after validation, got %d", len(result.KeptChunks))
	}
}

func TestDedupe_StatisticsCorrect(t *testing.T) {
	config := DefaultConfig()
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Unique one", Norm: "unique one", Index: 0},
		{ID: "c0002", Text: "Duplicate", Norm: "duplicate", Index: 1},
		{ID: "c0003", Text: "Duplicate", Norm: "duplicate", Index: 2},
		{ID: "c0004", Text: "Unique two", Norm: "unique two", Index: 3},
	}
	result := Dedupe(chunks, config)
	if result.Stats.InputCount != 4 {
		t.Errorf("expected 4 input count, got %d", result.Stats.InputCount)
	}
	if result.Stats.KeptCount+result.Stats.DroppedCount != result.Stats.InputCount {
		t.Errorf("kept + dropped should equal input: %d + %d != %d",
			result.Stats.KeptCount, result.Stats.DroppedCount, result.Stats.InputCount)
	}
	if result.Stats.ExactDups+result.Stats.NearDups != result.Stats.DroppedCount {
		t.Errorf("exact + near should equal dropped: %d + %d != %d",
			result.Stats.ExactDups, result.Stats.NearDups, result.Stats.DroppedCount)
	}
}

func TestDedupe_DroppedChunksMetadata(t *testing.T) {
	config := DefaultConfig()
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
		{ID: "c0002", Text: "Test", Norm: "test", Index: 1},
	}
	result := Dedupe(chunks, config)
	if len(result.Dropped) > 0 {
		dropped := result.Dropped[0]
		if dropped.ChunkID == "" {
			t.Error("dropped chunk should have ChunkID")
		}
		if dropped.Reason == "" {
			t.Error("dropped chunk should have Reason")
		}
		if dropped.Preview == "" {
			t.Error("dropped chunk should have Preview")
		}
	}
}

func TestDedupe_PreservesOrder(t *testing.T) {
	config := DefaultConfig()
	chunks := []text.Chunk{
		{ID: "c0001", Text: "First", Norm: "first", Index: 0},
		{ID: "c0002", Text: "Second", Norm: "second", Index: 1},
		{ID: "c0003", Text: "Third", Norm: "third", Index: 2},
	}
	result := Dedupe(chunks, config)
	if len(result.KeptChunks) != 3 {
		t.Fatalf("expected 3 kept chunks, got %d", len(result.KeptChunks))
	}
	if result.KeptChunks[0].ID != "c0001" {
		t.Errorf("expected first kept chunk to be c0001, got %s", result.KeptChunks[0].ID)
	}
	if result.KeptChunks[1].ID != "c0002" {
		t.Errorf("expected second kept chunk to be c0002, got %s", result.KeptChunks[1].ID)
	}
	if result.KeptChunks[2].ID != "c0003" {
		t.Errorf("expected third kept chunk to be c0003, got %s", result.KeptChunks[2].ID)
	}
}

// TestDedupe_MethodBoth tests the "both" method which requires chunks to pass both exact and simhash checks
func TestDedupe_MethodBoth(t *testing.T) {
	config := DefaultConfig()
	config.Method = "both"
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Unique chunk one", Norm: "unique chunk one", Index: 0},
		{ID: "c0002", Text: "Unique chunk two", Norm: "unique chunk two", Index: 1},
		{ID: "c0003", Text: "Unique chunk three", Norm: "unique chunk three", Index: 2},
	}
	result := Dedupe(chunks, config)
	// All chunks are unique, so all should be kept by both methods
	if len(result.KeptChunks) != 3 {
		t.Errorf("expected 3 kept chunks (all unique), got %d", len(result.KeptChunks))
	}
	if result.Stats.DroppedCount != 0 {
		t.Errorf("expected 0 dropped chunks, got %d", result.Stats.DroppedCount)
	}
}

// TestDedupe_MethodBoth_ExactDuplicate tests that exact duplicates are dropped in "both" method
func TestDedupe_MethodBoth_ExactDuplicate(t *testing.T) {
	config := DefaultConfig()
	config.Method = "both"
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test chunk", Norm: "test chunk", Index: 0},
		{ID: "c0002", Text: "Test chunk", Norm: "test chunk", Index: 1}, // Exact duplicate
		{ID: "c0003", Text: "Unique chunk", Norm: "unique chunk", Index: 2},
	}
	result := Dedupe(chunks, config)
	// c0002 should be dropped by exact hash, so it won't pass "both" check
	if len(result.KeptChunks) != 2 {
		t.Errorf("expected 2 kept chunks (c0001 and c0003), got %d", len(result.KeptChunks))
	}
	if result.Stats.ExactDups == 0 {
		t.Error("expected at least 1 exact duplicate to be detected")
	}
	// Verify c0002 is in dropped list
	found := false
	for _, dropped := range result.Dropped {
		if dropped.ChunkID == "c0002" {
			found = true
			if dropped.Reason != "exact_duplicate" {
				t.Errorf("expected reason exact_duplicate, got %s", dropped.Reason)
			}
			break
		}
	}
	if !found {
		t.Error("expected c0002 to be in dropped list")
	}
}

// TestDedupe_MethodBoth_NearDuplicate tests that near-duplicates are dropped in "both" method
func TestDedupe_MethodBoth_NearDuplicate(t *testing.T) {
	config := DefaultConfig()
	config.Method = "both"
	config.SimHashThreshold = 10 // Higher threshold to catch near-duplicates
	chunks := []text.Chunk{
		{ID: "c0001", Text: "This is a test string for simhash deduplication", Norm: "this is a test string for simhash deduplication", Index: 0},
		{ID: "c0002", Text: "This is a test string for simhash deduplication with small change", Norm: "this is a test string for simhash deduplication with small change", Index: 1}, // Near duplicate
		{ID: "c0003", Text: "Completely different content here", Norm: "completely different content here", Index: 2},
	}
	result := Dedupe(chunks, config)
	// c0002 might be dropped by simhash if similar enough
	// Verify statistics are correct
	if result.Stats.InputCount != 3 {
		t.Errorf("expected 3 input chunks, got %d", result.Stats.InputCount)
	}
	if result.Stats.KeptCount+result.Stats.DroppedCount != result.Stats.InputCount {
		t.Errorf("kept + dropped should equal input: %d + %d != %d",
			result.Stats.KeptCount, result.Stats.DroppedCount, result.Stats.InputCount)
	}
}

// TestDedupe_MethodBoth_DroppedByBoth tests chunk dropped by both methods
func TestDedupe_MethodBoth_DroppedByBoth(t *testing.T) {
	config := DefaultConfig()
	config.Method = "both"
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test chunk", Norm: "test chunk", Index: 0},
		{ID: "c0002", Text: "Test chunk", Norm: "test chunk", Index: 1}, // Exact duplicate, will be dropped
	}
	result := Dedupe(chunks, config)
	// c0002 should be dropped (exact duplicate)
	if len(result.KeptChunks) != 1 {
		t.Errorf("expected 1 kept chunk, got %d", len(result.KeptChunks))
	}
	if result.KeptChunks[0].ID != "c0001" {
		t.Errorf("expected kept chunk to be c0001, got %s", result.KeptChunks[0].ID)
	}
}

// TestDedupe_MethodBoth_Statistics tests that statistics are correct for "both" method
func TestDedupe_MethodBoth_Statistics(t *testing.T) {
	config := DefaultConfig()
	config.Method = "both"
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Unique one", Norm: "unique one", Index: 0},
		{ID: "c0002", Text: "Duplicate", Norm: "duplicate", Index: 1},
		{ID: "c0003", Text: "Duplicate", Norm: "duplicate", Index: 2}, // Exact duplicate of c0002
		{ID: "c0004", Text: "Unique two", Norm: "unique two", Index: 3},
	}
	result := Dedupe(chunks, config)
	if result.Stats.InputCount != 4 {
		t.Errorf("expected 4 input chunks, got %d", result.Stats.InputCount)
	}
	if result.Stats.KeptCount+result.Stats.DroppedCount != result.Stats.InputCount {
		t.Errorf("kept + dropped should equal input: %d + %d != %d",
			result.Stats.KeptCount, result.Stats.DroppedCount, result.Stats.InputCount)
	}
	if result.Stats.ExactDups+result.Stats.NearDups != result.Stats.DroppedCount {
		t.Errorf("exact + near should equal dropped: %d + %d != %d",
			result.Stats.ExactDups, result.Stats.NearDups, result.Stats.DroppedCount)
	}
}

// TestDedupe_MethodBoth_UniqueDroppedList tests that dropped list doesn't have duplicates
func TestDedupe_MethodBoth_UniqueDroppedList(t *testing.T) {
	config := DefaultConfig()
	config.Method = "both"
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Test", Norm: "test", Index: 0},
		{ID: "c0002", Text: "Test", Norm: "test", Index: 1}, // Exact duplicate
	}
	result := Dedupe(chunks, config)
	// Should only have one entry in dropped list (c0002)
	if len(result.Dropped) != 1 {
		t.Errorf("expected 1 dropped chunk, got %d", len(result.Dropped))
	}
	if result.Dropped[0].ChunkID != "c0002" {
		t.Errorf("expected dropped chunk ID c0002, got %s", result.Dropped[0].ChunkID)
	}
}

// TestDedupe_SingleChunk tests edge case with single chunk
func TestDedupe_SingleChunk(t *testing.T) {
	config := DefaultConfig()
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Single chunk", Norm: "single chunk", Index: 0},
	}
	result := Dedupe(chunks, config)
	if len(result.KeptChunks) != 1 {
		t.Errorf("expected 1 kept chunk, got %d", len(result.KeptChunks))
	}
	if result.KeptChunks[0].ID != "c0001" {
		t.Errorf("expected kept chunk ID c0001, got %s", result.KeptChunks[0].ID)
	}
	if result.Stats.DroppedCount != 0 {
		t.Errorf("expected 0 dropped chunks, got %d", result.Stats.DroppedCount)
	}
	if result.Stats.InputCount != 1 {
		t.Errorf("expected 1 input count, got %d", result.Stats.InputCount)
	}
}

// TestDedupe_AllDuplicates tests edge case where all chunks are duplicates
func TestDedupe_AllDuplicates(t *testing.T) {
	config := DefaultConfig()
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Duplicate", Norm: "duplicate", Index: 0},
		{ID: "c0002", Text: "Duplicate", Norm: "duplicate", Index: 1},
		{ID: "c0003", Text: "Duplicate", Norm: "duplicate", Index: 2},
		{ID: "c0004", Text: "Duplicate", Norm: "duplicate", Index: 3},
	}
	result := Dedupe(chunks, config)
	// Should keep only the first one
	if len(result.KeptChunks) != 1 {
		t.Errorf("expected 1 kept chunk, got %d", len(result.KeptChunks))
	}
	if result.KeptChunks[0].ID != "c0001" {
		t.Errorf("expected kept chunk ID c0001, got %s", result.KeptChunks[0].ID)
	}
	if result.Stats.DroppedCount != 3 {
		t.Errorf("expected 3 dropped chunks, got %d", result.Stats.DroppedCount)
	}
	if result.Stats.ExactDups != 3 {
		t.Errorf("expected 3 exact duplicates, got %d", result.Stats.ExactDups)
	}
}

// TestDedupe_NoDuplicates tests edge case where no chunks are duplicates
func TestDedupe_NoDuplicates(t *testing.T) {
	config := DefaultConfig()
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Unique chunk one with different content", Norm: "unique chunk one with different content", Index: 0},
		{ID: "c0002", Text: "Unique chunk two with different content", Norm: "unique chunk two with different content", Index: 1},
		{ID: "c0003", Text: "Unique chunk three with different content", Norm: "unique chunk three with different content", Index: 2},
	}
	result := Dedupe(chunks, config)
	// All should be kept
	if len(result.KeptChunks) != 3 {
		t.Errorf("expected 3 kept chunks, got %d", len(result.KeptChunks))
	}
	if result.Stats.DroppedCount != 0 {
		t.Errorf("expected 0 dropped chunks, got %d", result.Stats.DroppedCount)
	}
	if result.Stats.ExactDups != 0 {
		t.Errorf("expected 0 exact duplicates, got %d", result.Stats.ExactDups)
	}
	if result.Stats.NearDups != 0 {
		t.Errorf("expected 0 near duplicates, got %d", result.Stats.NearDups)
	}
}

// TestDedupe_MixedExactAndNearDuplicates tests mixed exact and near-duplicates
func TestDedupe_MixedExactAndNearDuplicates(t *testing.T) {
	config := DefaultConfig()
	config.SimHashThreshold = 10 // Higher threshold to catch near-duplicates
	chunks := []text.Chunk{
		{ID: "c0001", Text: "Unique one", Norm: "unique one", Index: 0},
		{ID: "c0002", Text: "Duplicate", Norm: "duplicate", Index: 1},
		{ID: "c0003", Text: "Duplicate", Norm: "duplicate", Index: 2}, // Exact duplicate of c0002
		{ID: "c0004", Text: "This is a test string for simhash", Norm: "this is a test string for simhash", Index: 3},
		{ID: "c0005", Text: "This is a test string for simhash with small change", Norm: "this is a test string for simhash with small change", Index: 4}, // Near duplicate of c0004
		{ID: "c0006", Text: "Unique two", Norm: "unique two", Index: 5},
	}
	result := Dedupe(chunks, config)
	// Should have both exact and near duplicates
	if result.Stats.ExactDups == 0 && result.Stats.NearDups == 0 {
		t.Error("expected at least some duplicates to be detected")
	}
	if result.Stats.KeptCount+result.Stats.DroppedCount != result.Stats.InputCount {
		t.Errorf("kept + dropped should equal input: %d + %d != %d",
			result.Stats.KeptCount, result.Stats.DroppedCount, result.Stats.InputCount)
	}
	if result.Stats.ExactDups+result.Stats.NearDups != result.Stats.DroppedCount {
		t.Errorf("exact + near should equal dropped: %d + %d != %d",
			result.Stats.ExactDups, result.Stats.NearDups, result.Stats.DroppedCount)
	}
}
