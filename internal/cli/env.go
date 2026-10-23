// Package cli implements the peppol-sh commands.
package cli

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Env is everything a run of the CLI takes from the process, so that tests can
// replace it.
type Env struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Getenv func(string) string
	// ConfigDir holds config.toml. Empty when the OS has no config directory.
	ConfigDir string
	Transport http.RoundTripper
	// Sleep waits d, or returns the error of ctx when ctx ends first.
	Sleep func(ctx context.Context, d time.Duration) error
}

// Main runs the CLI with the real process environment and returns the exit code.
func Main() int {
	configDir := ""
	if dir, err := os.UserConfigDir(); err == nil {
		configDir = filepath.Join(dir, "peppol-sh")
	}
	return Run(context.Background(), os.Args[1:], Env{
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Getenv:    os.Getenv,
		ConfigDir: configDir,
		Transport: http.DefaultTransport,
		Sleep:     sleep,
	})
}
