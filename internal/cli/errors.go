package cli

import (
	"errors"
	"fmt"
	"io"
)

const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
	exitAuth  = 3
)

// cliError is an error that knows its exit code and how to render itself.
// Every error that a command returns is one; any other error comes from cobra
// and is a usage error.
type cliError struct {
	code int
	msg  string
	// extra lines after "Error: msg", for example a request ID or a hint.
	lines []string
}

func (e *cliError) Error() string { return e.msg }

func newError(code int, format string, args ...any) *cliError {
	return &cliError{code: code, msg: fmt.Sprintf(format, args...)}
}

func (e *cliError) with(line string) *cliError {
	e.lines = append(e.lines, line)
	return e
}

func render(w io.Writer, err error, usageHint string) int {
	var ce *cliError
	if !errors.As(err, &ce) {
		ce = newError(exitUsage, "%s", err.Error()).with(usageHint)
	}
	fmt.Fprintf(w, "Error: %s\n", ce.msg)
	for _, line := range ce.lines {
		fmt.Fprintln(w, line)
	}
	return ce.code
}
