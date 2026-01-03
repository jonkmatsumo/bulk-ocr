package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunner_LookPath(t *testing.T) {
	r := New()

	// Test finding a common binary
	path, err := r.LookPath("sh")
	if err != nil {
		t.Fatalf("LookPath failed: %v", err)
	}
	if path == "" {
		t.Error("expected path, got empty string")
	}

	// Test non-existent binary
	_, err = r.LookPath("nonexistent-binary-xyz123")
	if err == nil {
		t.Error("expected error for non-existent binary")
	}
}

func TestRunner_Run_Success(t *testing.T) {
	r := New()
	ctx := context.Background()

	result, err := r.Run(ctx, "echo", []string{"hello", "world"}, RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "hello") || !strings.Contains(result.Stdout, "world") {
		t.Errorf("expected stdout to contain 'hello world', got: %s", result.Stdout)
	}

	if result.Cmd == "" {
		t.Error("expected Cmd to be set")
	}

	if result.DurationMs <= 0 {
		t.Error("expected DurationMs to be positive")
	}
}

func TestRunner_Run_NonZeroExit(t *testing.T) {
	r := New()
	ctx := context.Background()

	result, err := r.Run(ctx, "sh", []string{"-c", "exit 42"}, RunOpts{})
	if err == nil {
		t.Fatal("expected error for non-zero exit")
	}

	if result.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", result.ExitCode)
	}

	execErr, ok := err.(*ExecError)
	if !ok {
		t.Fatalf("expected ExecError, got %T", err)
	}

	if execErr.Bin != "sh" {
		t.Errorf("expected Bin to be 'sh', got %s", execErr.Bin)
	}

	if execErr.Result.ExitCode != 42 {
		t.Errorf("expected ExecError.Result.ExitCode to be 42, got %d", execErr.Result.ExitCode)
	}
}

func TestRunner_Run_Timeout(t *testing.T) {
	r := New()
	ctx := context.Background()

	opts := RunOpts{
		Timeout: 200 * time.Millisecond,
	}

	result, err := r.Run(ctx, "sh", []string{"-c", "sleep 5"}, opts)
	if err == nil {
		t.Fatal("expected error for timeout")
	}

	// Check for timeout-related errors
	errStr := err.Error()
	hasTimeout := strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "timed out") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context deadline exceeded")
	if !hasTimeout {
		t.Errorf("expected timeout error, got: %v", err)
	}

	// Process should have been killed, so exit code might be non-zero or -1
	if result.ExitCode == 0 {
		t.Log("warning: exit code is 0 after timeout (process may have been killed before exit)")
	}
}

func TestRunner_Run_ContextCancellation(t *testing.T) {
	r := New()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	opts := RunOpts{
		Timeout: 5 * time.Second, // Should be canceled before timeout
	}

	result, err := r.Run(ctx, "sh", []string{"-c", "sleep 1"}, opts)
	if err == nil {
		t.Fatal("expected error for cancellation")
	}

	if !strings.Contains(err.Error(), "cancel") {
		t.Errorf("expected cancellation error, got: %v", err)
	}

	_ = result // Result may be incomplete
}

func TestRunner_Run_CaptureTruncation(t *testing.T) {
	r := New()
	ctx := context.Background()

	opts := RunOpts{
		MaxCaptureBytes: 2 * 1024 * 1024, // 2MB limit
		StdoutMode:      Capture,
	}

	// Use a command that generates large output without passing it as argument
	// Use yes command or generate via a loop to avoid argument list limits
	result, err := r.Run(ctx, "sh", []string{"-c", "yes x | head -c 3145728"}, opts) // 3MB
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "...[truncated]") {
		t.Error("expected truncation marker in stdout")
	}

	if len(result.Stdout) > opts.MaxCaptureBytes+1000 { // Allow some overhead
		t.Errorf("captured output too large: %d bytes (limit: %d)", len(result.Stdout), opts.MaxCaptureBytes)
	}
}

func TestRunner_Run_Streaming(t *testing.T) {
	r := New()
	ctx := context.Background()

	opts := RunOpts{
		StdoutMode: StreamAndCapture,
		StderrMode: StreamAndCapture,
	}

	result, err := r.Run(ctx, "echo", []string{"streamed"}, opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have captured output
	if !strings.Contains(result.Stdout, "streamed") {
		t.Errorf("expected captured stdout, got: %s", result.Stdout)
	}
}

func TestRunner_Run_DiscardOutput(t *testing.T) {
	r := New()
	ctx := context.Background()

	opts := RunOpts{
		StdoutMode: Discard,
		StderrMode: Discard,
	}

	result, err := r.Run(ctx, "echo", []string{"discarded"}, opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Output should be empty
	if result.Stdout != "" {
		t.Errorf("expected empty stdout, got: %s", result.Stdout)
	}
}

func TestRunner_Run_WorkingDirectory(t *testing.T) {
	r := New()
	ctx := context.Background()

	tmpDir := t.TempDir()

	opts := RunOpts{
		Dir: tmpDir,
	}

	result, err := r.Run(ctx, "pwd", []string{}, opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !strings.Contains(result.Stdout, tmpDir) {
		t.Errorf("expected working directory %s in output, got: %s", tmpDir, result.Stdout)
	}
}

func TestRunner_Run_EnvironmentVariables(t *testing.T) {
	r := New()
	ctx := context.Background()

	opts := RunOpts{
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	result, err := r.Run(ctx, "sh", []string{"-c", "echo $TEST_VAR"}, opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "test_value") {
		t.Errorf("expected environment variable in output, got: %s", result.Stdout)
	}
}

func TestRunner_Run_CommandFormatting(t *testing.T) {
	r := New()
	ctx := context.Background()

	result, err := r.Run(ctx, "echo", []string{"hello world", "with spaces"}, RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Command should be properly formatted with quotes
	if !strings.Contains(result.Cmd, `"hello world"`) {
		t.Errorf("expected quoted args in Cmd, got: %s", result.Cmd)
	}

	// Should be copy/pasteable
	if strings.Contains(result.Cmd, "\n") {
		t.Error("Cmd should not contain newlines")
	}
}

func TestRunner_Run_EmptyArgs(t *testing.T) {
	r := New()
	ctx := context.Background()

	result, err := r.Run(ctx, "echo", []string{}, RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.Cmd == "" {
		t.Error("expected Cmd to be set even with empty args")
	}

	if !strings.Contains(result.Cmd, "echo") {
		t.Errorf("expected 'echo' in Cmd, got: %s", result.Cmd)
	}
}

func TestRunner_Run_StderrCapture(t *testing.T) {
	r := New()
	ctx := context.Background()

	opts := RunOpts{
		StderrMode: Capture,
	}

	result, err := r.Run(ctx, "sh", []string{"-c", "echo 'error' >&2"}, opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !strings.Contains(result.Stderr, "error") {
		t.Errorf("expected stderr capture, got: %s", result.Stderr)
	}
}

func TestExecError_Unwrap(t *testing.T) {
	r := New()
	ctx := context.Background()

	_, err := r.Run(ctx, "sh", []string{"-c", "exit 1"}, RunOpts{})
	if err == nil {
		t.Fatal("expected error")
	}

	execErr, ok := err.(*ExecError)
	if !ok {
		t.Fatalf("expected ExecError, got %T", err)
	}

	unwrapped := execErr.Unwrap()
	if unwrapped == nil {
		t.Error("expected unwrapped error to be non-nil")
	}
}

func TestRunner_Run_DefaultOutputModes(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Test with zero-value RunOpts (should default to StreamAndCapture)
	result, err := r.Run(ctx, "echo", []string{"test"}, RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have captured output
	if result.Stdout == "" {
		t.Error("expected captured stdout with default mode")
	}
}

func TestRunner_Run_NonexistentBinary(t *testing.T) {
	r := New()
	ctx := context.Background()

	_, err := r.Run(ctx, "nonexistent-binary-xyz123", []string{}, RunOpts{})
	if err == nil {
		t.Fatal("expected error for non-existent binary")
	}
}

func TestLimitedWriter(t *testing.T) {
	w := newLimitedWriter(100)

	// Write more than limit
	data := make([]byte, 200)
	for i := range data {
		data[i] = 'x'
	}

	n, err := w.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != 200 {
		t.Errorf("expected to report 200 bytes written, got %d", n)
	}

	s := w.String()
	if !strings.Contains(s, "...[truncated]") {
		t.Error("expected truncation marker")
	}

	if len(s) > 150 { // Allow some overhead
		t.Errorf("output too large: %d bytes", len(s))
	}
}

func TestFormatCommand(t *testing.T) {
	tests := []struct {
		name string
		bin  string
		args []string
		want string
	}{
		{
			name: "simple",
			bin:  "echo",
			args: []string{"hello"},
			want: "echo hello",
		},
		{
			name: "with spaces",
			bin:  "echo",
			args: []string{"hello world"},
			want: `echo "hello world"`,
		},
		{
			name: "empty arg",
			bin:  "echo",
			args: []string{""},
			want: `echo ""`,
		},
		{
			name: "multiple args",
			bin:  "sh",
			args: []string{"-c", "echo test"},
			want: `sh -c "echo test"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCommand(tt.bin, tt.args)
			if got != tt.want {
				t.Errorf("formatCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test that runner works with real commands that might be in the environment
func TestRunner_Run_RealCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real command tests in short mode")
	}

	r := New()
	ctx := context.Background()

	// Test with commands that should exist
	commands := []struct {
		bin  string
		args []string
	}{
		{"sh", []string{"-c", "true"}},
		{"echo", []string{"test"}},
	}

	for _, cmd := range commands {
		if _, err := r.LookPath(cmd.bin); err != nil {
			t.Logf("skipping %s (not in PATH)", cmd.bin)
			continue
		}

		result, err := r.Run(ctx, cmd.bin, cmd.args, RunOpts{})
		if err != nil {
			t.Errorf("Run(%s) failed: %v", cmd.bin, err)
			continue
		}

		if result.ExitCode != 0 {
			t.Errorf("Run(%s) returned exit code %d", cmd.bin, result.ExitCode)
		}
	}
}
