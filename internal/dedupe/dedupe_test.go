package dedupe

import "testing"

func TestDedupe(t *testing.T) {
	input := []string{"chunk1", "chunk2", "chunk3"}
	result := Dedupe(input)

	if len(result) != len(input) {
		t.Errorf("expected %d chunks, got %d", len(input), len(result))
	}

	for i := range input {
		if result[i] != input[i] {
			t.Errorf("position %d: expected %s, got %s", i, input[i], result[i])
		}
	}
}

func TestDedupe_Empty(t *testing.T) {
	result := Dedupe([]string{})
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d elements", len(result))
	}
}
