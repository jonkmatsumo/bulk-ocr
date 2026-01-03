package text

import "testing"

func TestChunk(t *testing.T) {
	result := Chunk("test text")
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}
