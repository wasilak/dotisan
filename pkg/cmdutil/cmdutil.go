// Package cmdutil provides utilities for running external commands.
//
// This package is used by package providers (BrewProvider, NpmProvider, etc.)
// to execute system commands with consistent error handling and output capture.
package cmdutil

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// RunOptions contains options for running a command.
type RunOptions struct {
	// Timeout for the command (0 = no timeout)
	Timeout time.Duration

	// Working directory for the command
	Dir string

	// Environment variables to add
	Env []string
}

// Result contains the result of running a command.
type Result struct {
	// Stdout is the command's standard output
	Stdout string

	// Stderr is the command's standard error
	Stderr string

	// ExitCode is the command's exit code
	ExitCode int

	// Error is any error that occurred
	Error error
}

// Run executes a command and returns the result.
// If the command fails, the error is captured in Result.Error.
func Run(ctx context.Context, name string, args []string, opts *RunOptions) Result {
	if opts == nil {
		opts = &RunOptions{}
	}

	// Create command
	cmd := exec.CommandContext(ctx, name, args...)

	// Set working directory
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	// Set environment
	if len(opts.Env) > 0 {
		cmd.Env = append(cmd.Environ(), opts.Env...)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()

	result := Result{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: 0,
		Error:    err,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
	}

	return result
}

// RunSimple is a convenience function for simple command execution.
// Returns stdout, stderr, and error.
func RunSimple(ctx context.Context, name string, args ...string) (stdout, stderr string, err error) {
	result := Run(ctx, name, args, nil)
	return result.Stdout, result.Stderr, result.Error
}

// RunSimpleFn is the function used by callers to execute simple commands.
// Tests may override this to simulate command execution.
var RunSimpleFn = RunSimple

// CheckExecutable checks if an executable exists in PATH.
// Returns the path to the executable if found, empty string if not.
func CheckExecutable(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}
