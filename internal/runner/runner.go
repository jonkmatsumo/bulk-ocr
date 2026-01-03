package runner

import (
	"context"
	"fmt"
)

// Runner executes external commands.
type Runner struct{}

// Run executes an external command and returns its output.
// This is a placeholder implementation that returns "not implemented".
func (r *Runner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
