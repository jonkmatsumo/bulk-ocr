package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// OutputMode controls how command output is handled.
type OutputMode int

const (
	// Stream outputs to stdout/stderr in real-time.
	Stream OutputMode = iota
	// Capture outputs to memory (for Result).
	Capture
	// Discard outputs completely.
	Discard
	// StreamAndCapture both streams and captures output.
	StreamAndCapture
)

// RunOpts configures command execution.
type RunOpts struct {
	// Dir is the working directory (optional).
	Dir string
	// Env is extra environment variables (merged with current env).
	Env map[string]string
	// Timeout is the maximum execution time (0 means no timeout).
	Timeout time.Duration
	// StdoutMode controls stdout handling.
	StdoutMode OutputMode
	// StderrMode controls stderr handling.
	StderrMode OutputMode
	// MaxCaptureBytes limits captured output to prevent OOM (default 2MB).
	MaxCaptureBytes int
}

// Result contains the result of a command execution.
type Result struct {
	// Cmd is a printable, copy/pasteable command line (escaped).
	Cmd string
	// ExitCode is the process exit code.
	ExitCode int
	// DurationMs is execution duration in milliseconds.
	DurationMs int64
	// Stdout is captured stdout (may be truncated).
	Stdout string
	// Stderr is captured stderr (may be truncated).
	Stderr string
}

// ExecError represents a command execution error.
type ExecError struct {
	// Bin is the binary name.
	Bin string
	// Args are the command arguments.
	Args []string
	// Result contains the execution result.
	Result Result
	// Cause is the underlying error (if any).
	Cause error
}

func (e *ExecError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("command failed: %s (exit code %d): %v", e.Result.Cmd, e.Result.ExitCode, e.Cause)
	}
	return fmt.Sprintf("command failed: %s (exit code %d)", e.Result.Cmd, e.Result.ExitCode)
}

func (e *ExecError) Unwrap() error {
	return e.Cause
}

// Runner executes external commands.
type Runner struct{}

// New creates a new Runner.
func New() *Runner {
	return &Runner{}
}

// LookPath finds the binary in PATH.
func (r *Runner) LookPath(bin string) (string, error) {
	return exec.LookPath(bin)
}

// Run executes an external command with the given options.
func (r *Runner) Run(ctx context.Context, bin string, args []string, opts RunOpts) (Result, error) {
	start := time.Now()

	// Apply defaults
	if opts.StdoutMode == 0 {
		opts.StdoutMode = StreamAndCapture
	}
	if opts.StderrMode == 0 {
		opts.StderrMode = StreamAndCapture
	}
	if opts.MaxCaptureBytes == 0 {
		opts.MaxCaptureBytes = 2 * 1024 * 1024 // 2MB default
	}

	// Apply timeout if specified (before creating command)
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Create command with context for cancellation
	cmd := exec.CommandContext(ctx, bin, args...)

	// Set working directory
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	// Set environment variables
	if len(opts.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range opts.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Format command string for Result.Cmd
	cmdStr := formatCommand(bin, args)

	// Setup stdout
	var stdoutWriter io.Writer
	var stdoutCapture *limitedWriter
	if opts.StdoutMode == Capture || opts.StdoutMode == StreamAndCapture {
		stdoutCapture = newLimitedWriter(opts.MaxCaptureBytes)
		stdoutWriter = stdoutCapture
	}
	if opts.StdoutMode == Stream || opts.StdoutMode == StreamAndCapture {
		if stdoutWriter == nil {
			stdoutWriter = os.Stdout
		} else {
			stdoutWriter = io.MultiWriter(stdoutWriter, os.Stdout)
		}
	}
	if stdoutWriter == nil {
		stdoutWriter = io.Discard
	}
	cmd.Stdout = stdoutWriter

	// Setup stderr
	var stderrWriter io.Writer
	var stderrCapture *limitedWriter
	if opts.StderrMode == Capture || opts.StderrMode == StreamAndCapture {
		stderrCapture = newLimitedWriter(opts.MaxCaptureBytes)
		stderrWriter = stderrCapture
	}
	if opts.StderrMode == Stream || opts.StderrMode == StreamAndCapture {
		if stderrWriter == nil {
			stderrWriter = os.Stderr
		} else {
			stderrWriter = io.MultiWriter(stderrWriter, os.Stderr)
		}
	}
	if stderrWriter == nil {
		stderrWriter = io.Discard
	}
	cmd.Stderr = stderrWriter

	// Execute command
	err := cmd.Run()
	duration := time.Since(start)

	// Build result
	result := Result{
		Cmd:        cmdStr,
		ExitCode:   cmd.ProcessState.ExitCode(),
		DurationMs: duration.Milliseconds(),
	}

	// Capture output
	if stdoutCapture != nil {
		result.Stdout = stdoutCapture.String()
	}
	if stderrCapture != nil {
		result.Stderr = stderrCapture.String()
	}

	// Handle errors
	if err != nil {
		// Check for context errors
		if ctx.Err() == context.DeadlineExceeded {
			return result, fmt.Errorf("command timed out after %v: %w", opts.Timeout, err)
		}
		if ctx.Err() == context.Canceled {
			return result, fmt.Errorf("command canceled: %w", err)
		}

		// Non-zero exit code
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() != 0 {
			return result, &ExecError{
				Bin:    bin,
				Args:   args,
				Result: result,
				Cause:  err,
			}
		}

		// Other errors
		return result, fmt.Errorf("command execution failed: %w", err)
	}

	return result, nil
}

// formatCommand formats a command and args as a safe, copy/pasteable string.
func formatCommand(bin string, args []string) string {
	var parts []string
	parts = append(parts, quoteArg(bin))
	for _, arg := range args {
		parts = append(parts, quoteArg(arg))
	}
	return strings.Join(parts, " ")
}

// quoteArg quotes an argument if it contains spaces or special characters.
func quoteArg(arg string) string {
	if arg == "" {
		return `""`
	}
	// Simple check: if contains space, quote it
	if strings.ContainsAny(arg, " \t\n\"'") {
		return fmt.Sprintf(`"%s"`, strings.ReplaceAll(arg, `"`, `\"`))
	}
	return arg
}

// limitedWriter limits the amount of data written and appends truncation message.
type limitedWriter struct {
	maxBytes  int
	buf       strings.Builder
	written   int
	truncated bool
}

func newLimitedWriter(maxBytes int) *limitedWriter {
	return &limitedWriter{
		maxBytes: maxBytes,
	}
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	remaining := w.maxBytes - w.written
	if remaining <= 0 {
		if !w.truncated {
			w.truncated = true
		}
		return len(p), nil // Discard but report as written
	}

	toWrite := p
	if len(toWrite) > remaining {
		toWrite = toWrite[:remaining]
		w.truncated = true
	}

	n, err := w.buf.Write(toWrite)
	w.written += n
	return len(p), err // Report original length
}

func (w *limitedWriter) String() string {
	s := w.buf.String()
	if w.truncated {
		s += "\n...[truncated]"
	}
	return s
}
