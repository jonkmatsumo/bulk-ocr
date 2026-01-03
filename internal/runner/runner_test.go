package runner

import (
	"context"
	"testing"
)

func TestRunner_Run(t *testing.T) {
	r := &Runner{}
	ctx := context.Background()

	output, err := r.Run(ctx, "echo", "test")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if output != nil {
		t.Errorf("expected nil output, got %v", output)
	}
	if err.Error() != "not implemented" {
		t.Errorf("expected 'not implemented' error, got %v", err)
	}
}

func TestRunner_Run_ContextCancellation(t *testing.T) {
	r := &Runner{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Even though context is cancelled, we expect "not implemented" error
	// This test ensures the function signature accepts context for future use
	_, err := r.Run(ctx, "echo", "test")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

