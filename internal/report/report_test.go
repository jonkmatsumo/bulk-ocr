package report

import (
	"encoding/json"
	"testing"
)

func TestReport_JSONMarshaling(t *testing.T) {
	r := Report{
		InputImages: 42,
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("failed to marshal report: %v", err)
	}

	var unmarshaled Report
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal report: %v", err)
	}

	if unmarshaled.InputImages != r.InputImages {
		t.Errorf("expected %d input images, got %d", r.InputImages, unmarshaled.InputImages)
	}
}
