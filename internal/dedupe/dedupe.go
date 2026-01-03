package dedupe

import (
	"crypto/sha1"
	"fmt"
	"math/bits"

	"github.com/jonkmatsumo/bulk-ocr/internal/text"
)

// DedupeResult contains the deduplicated chunks and metadata.
type DedupeResult struct {
	KeptChunks []text.Chunk
	Dropped    []DroppedChunk
	Stats      Stats
}

// DroppedChunk represents a chunk that was removed during deduplication.
type DroppedChunk struct {
	ChunkID        string // Original chunk ID (e.g., "c0005")
	Reason         string // "exact_duplicate" or "near_duplicate"
	MatchedChunkID string // ID of chunk it matched (if near-duplicate)
	Distance       int    // Hamming distance (if near-duplicate, 0 if exact)
	Preview        string // Truncated text preview (200 chars max)
}

// Stats contains deduplication statistics.
type Stats struct {
	InputCount   int
	KeptCount    int
	DroppedCount int
	ExactDups    int
	NearDups     int
}

// Config holds deduplication configuration.
type Config struct {
	Method           string // "exact", "simhash", or "both" (default: "simhash")
	SimHashK         int    // Character k-gram size (default: 5)
	SimHashThreshold int    // Hamming distance threshold (default: 6)
	Window           int    // Sliding window size (default: 250)
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		Method:           "simhash",
		SimHashK:         5,
		SimHashThreshold: 6,
		Window:           250,
	}
}

// Validate ensures config values are within valid ranges and sets defaults if needed.
func (c *Config) Validate() {
	if c.SimHashK <= 0 {
		c.SimHashK = 5
	}
	if c.SimHashThreshold < 0 {
		c.SimHashThreshold = 6
	}
	if c.SimHashThreshold > 64 {
		c.SimHashThreshold = 64
	}
	if c.Window < 0 {
		c.Window = 250
	}
	if c.Method != "exact" && c.Method != "simhash" && c.Method != "both" {
		c.Method = "simhash"
	}
}

// exactHashDedupe removes exact duplicates using SHA1 hash of normalized text.
func exactHashDedupe(chunks []text.Chunk) ([]text.Chunk, []DroppedChunk) {
	if len(chunks) == 0 {
		return []text.Chunk{}, []DroppedChunk{}
	}

	seen := make(map[string]string) // hash -> chunk ID
	var kept []text.Chunk
	var dropped []DroppedChunk

	for _, chunk := range chunks {
		// Handle empty normalized text
		if chunk.Norm == "" {
			// Keep empty chunks (edge case, shouldn't happen after normalization)
			kept = append(kept, chunk)
			continue
		}

		// Compute SHA1 hash of normalized text
		hash := sha1.Sum([]byte(chunk.Norm))
		hashStr := fmt.Sprintf("%x", hash)

		// Check if we've seen this hash before
		if existingID, exists := seen[hashStr]; exists {
			// This is an exact duplicate
			preview := chunk.Text
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			dropped = append(dropped, DroppedChunk{
				ChunkID:        chunk.ID,
				Reason:         "exact_duplicate",
				MatchedChunkID: existingID,
				Distance:       0,
				Preview:        preview,
			})
		} else {
			// First occurrence, keep it
			seen[hashStr] = chunk.ID
			kept = append(kept, chunk)
		}
	}

	return kept, dropped
}

// generateKgrams generates character k-grams from text.
func generateKgrams(text string, k int) []string {
	if k <= 0 || len(text) < k {
		return []string{}
	}

	var kgrams []string
	for i := 0; i <= len(text)-k; i++ {
		kgrams = append(kgrams, text[i:i+k])
	}

	return kgrams
}

// FNV-1a constants for 64-bit hashing
const (
	fnvOffsetBasis64 uint64 = 14695981039346656037
	fnvPrime64       uint64 = 1099511628211
)

// fnv1a64 computes FNV-1a 64-bit hash of data.
func fnv1a64(data []byte) uint64 {
	hash := fnvOffsetBasis64
	for _, b := range data {
		hash ^= uint64(b)
		hash *= fnvPrime64
	}
	return hash
}

// simhash64 computes SimHash signature for text using k-grams.
func simhash64(text string, k int) uint64 {
	if text == "" || k <= 0 {
		return 0
	}

	kgrams := generateKgrams(text, k)
	if len(kgrams) == 0 {
		return 0
	}

	// Initialize 64-element vector (one per bit position)
	vector := make([]int, 64)

	// Process each k-gram
	for _, kg := range kgrams {
		hash := fnv1a64([]byte(kg))
		// For each bit position, increment or decrement vector
		for i := 0; i < 64; i++ {
			if hash&(1<<i) != 0 {
				vector[i]++
			} else {
				vector[i]--
			}
		}
	}

	// Generate signature: set bit if vector[i] > 0
	var signature uint64
	for i := 0; i < 64; i++ {
		if vector[i] > 0 {
			signature |= 1 << i
		}
	}

	return signature
}

// hammingDistance computes Hamming distance between two 64-bit values.
func hammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// simhashDedupe removes near-duplicates using SimHash with sliding window.
func simhashDedupe(chunks []text.Chunk, config Config) ([]text.Chunk, []DroppedChunk) {
	if len(chunks) == 0 {
		return []text.Chunk{}, []DroppedChunk{}
	}

	// Pre-compute SimHash signatures for all chunks
	signatures := make([]uint64, len(chunks))
	for i, chunk := range chunks {
		signatures[i] = simhash64(chunk.Norm, config.SimHashK)
	}

	var kept []text.Chunk
	var keptSignatures []uint64 // Parallel array for signatures
	var dropped []DroppedChunk

	// Sliding window: maintain last N kept chunks
	windowSize := config.Window
	if windowSize == 0 {
		windowSize = len(chunks) // Compare against all if window is 0
	}

	for i, chunk := range chunks {
		sig := signatures[i]
		matched := false
		var matchedChunkID string
		minDistance := 65 // Larger than max possible (64)

		// Compare with chunks in sliding window
		windowStart := 0
		if len(kept) > windowSize {
			windowStart = len(kept) - windowSize
		}

		for j := windowStart; j < len(kept); j++ {
			dist := hammingDistance(sig, keptSignatures[j])
			if dist <= config.SimHashThreshold && dist < minDistance {
				matched = true
				matchedChunkID = kept[j].ID
				minDistance = dist
			}
		}

		if matched {
			// Near-duplicate found
			preview := chunk.Text
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			dropped = append(dropped, DroppedChunk{
				ChunkID:        chunk.ID,
				Reason:         "near_duplicate",
				MatchedChunkID: matchedChunkID,
				Distance:       minDistance,
				Preview:        preview,
			})
		} else {
			// Keep this chunk
			kept = append(kept, chunk)
			keptSignatures = append(keptSignatures, sig)
			// Window size is maintained by adjusting windowStart in the comparison loop above
		}
	}

	return kept, dropped
}

// Dedupe removes duplicates from chunks based on the configuration.
func Dedupe(chunks []text.Chunk, config Config) DedupeResult {
	config.Validate()

	if len(chunks) == 0 {
		return DedupeResult{
			KeptChunks: []text.Chunk{},
			Dropped:    []DroppedChunk{},
			Stats: Stats{
				InputCount:   0,
				KeptCount:    0,
				DroppedCount: 0,
				ExactDups:    0,
				NearDups:     0,
			},
		}
	}

	var kept []text.Chunk
	var dropped []DroppedChunk

	switch config.Method {
	case "exact":
		kept, dropped = exactHashDedupe(chunks)
	case "simhash":
		// Run exact hash pre-check first (fast path)
		exactKept, exactDropped := exactHashDedupe(chunks)
		// Then run SimHash on remaining chunks
		simhashKept, simhashDropped := simhashDedupe(exactKept, config)
		kept = simhashKept
		dropped = append(dropped, exactDropped...)
		dropped = append(dropped, simhashDropped...)
	case "both":
		// Run both methods independently and combine
		exactKept, exactDropped := exactHashDedupe(chunks)
		simhashKept, simhashDropped := simhashDedupe(chunks, config)
		// Combine: keep chunks that are kept by both methods
		// This is more conservative - only keep if not duplicate by either method
		exactKeptMap := make(map[string]bool)
		for _, c := range exactKept {
			exactKeptMap[c.ID] = true
		}
		simhashKeptMap := make(map[string]bool)
		for _, c := range simhashKept {
			simhashKeptMap[c.ID] = true
		}
		// Keep only chunks that pass both checks
		var bothKept []text.Chunk
		for _, chunk := range chunks {
			if exactKeptMap[chunk.ID] && simhashKeptMap[chunk.ID] {
				bothKept = append(bothKept, chunk)
			}
		}
		// Build dropped list from both methods
		allDropped := append(exactDropped, simhashDropped...)
		// Remove duplicates from dropped list (same chunk might be dropped by both)
		droppedMap := make(map[string]DroppedChunk)
		for _, d := range allDropped {
			if existing, exists := droppedMap[d.ChunkID]; !exists || d.Distance < existing.Distance {
				droppedMap[d.ChunkID] = d
			}
		}
		var uniqueDropped []DroppedChunk
		for _, d := range droppedMap {
			uniqueDropped = append(uniqueDropped, d)
		}
		kept = bothKept
		dropped = uniqueDropped
	default:
		// Default to simhash
		exactKept, exactDropped := exactHashDedupe(chunks)
		simhashKept, simhashDropped := simhashDedupe(exactKept, config)
		kept = simhashKept
		dropped = append(dropped, exactDropped...)
		dropped = append(dropped, simhashDropped...)
	}

	// Count statistics
	exactCount := 0
	nearCount := 0
	for _, d := range dropped {
		switch d.Reason {
		case "exact_duplicate":
			exactCount++
		case "near_duplicate":
			nearCount++
		}
	}

	return DedupeResult{
		KeptChunks: kept,
		Dropped:    dropped,
		Stats: Stats{
			InputCount:   len(chunks),
			KeptCount:    len(kept),
			DroppedCount: len(dropped),
			ExactDups:    exactCount,
			NearDups:     nearCount,
		},
	}
}
