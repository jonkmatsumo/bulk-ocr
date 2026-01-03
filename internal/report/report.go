package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jonkmatsumo/bulk-ocr/internal/dedupe"
)

// Report contains deduplication report data.
type Report struct {
	InputImages     int                   `json:"input_images"`
	InputChunks     int                   `json:"input_chunks"`
	KeptChunks      int                   `json:"kept_chunks"`
	DroppedChunks   int                   `json:"dropped_chunks"`
	ExactDuplicates int                   `json:"exact_duplicates"`
	NearDuplicates  int                   `json:"near_duplicates"`
	Config          Config                `json:"config"`
	Dropped         []dedupe.DroppedChunk `json:"dropped"`
	Timestamp       string                `json:"timestamp"`
}

// Config holds deduplication configuration for the report.
type Config struct {
	Method           string `json:"method"`
	SimHashK         int    `json:"simhash_k"`
	SimHashThreshold int    `json:"simhash_threshold"`
	Window           int    `json:"window"`
}

// WriteReport writes a deduplication report to a JSON file.
func WriteReport(result dedupe.DedupeResult, inputImages int, config dedupe.Config, path string) error {
	report := Report{
		InputImages:     inputImages,
		InputChunks:     result.Stats.InputCount,
		KeptChunks:      result.Stats.KeptCount,
		DroppedChunks:   result.Stats.DroppedCount,
		ExactDuplicates: result.Stats.ExactDups,
		NearDuplicates:  result.Stats.NearDups,
		Config: Config{
			Method:           config.Method,
			SimHashK:         config.SimHashK,
			SimHashThreshold: config.SimHashThreshold,
			Window:           config.Window,
		},
		Dropped:   result.Dropped,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close report file %s: %v\n", path, cerr)
		}
	}()

	if _, err := file.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return nil
}
