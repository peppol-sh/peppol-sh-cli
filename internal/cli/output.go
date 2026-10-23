package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/term"
)

func isTerminal(v any) bool {
	f, ok := v.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// palette adds ANSI colour only when stdout is a terminal and NO_COLOR is unset.
type palette struct{ on bool }

func (p palette) paint(code, s string) string {
	if !p.on {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (p palette) bold(s string) string { return p.paint("1", s) }

func (p palette) state(s string, good bool) string {
	if good {
		return p.paint("32", s)
	}
	return p.paint("31", s)
}

// writeRaw prints an API response body unchanged.
func (a *app) writeRaw(body []byte) error {
	if _, err := a.env.Stdout.Write(body); err != nil {
		return err
	}
	if isTerminal(a.env.Stdout) && !bytes.HasSuffix(body, []byte("\n")) {
		_, err := io.WriteString(a.env.Stdout, "\n")
		return err
	}
	return nil
}

func (a *app) writeJSON(v any) error {
	return json.NewEncoder(a.env.Stdout).Encode(v)
}

type field struct{ name, value string }

func (a *app) printFields(fields ...field) error {
	tw := tabwriter.NewWriter(a.env.Stdout, 0, 0, 2, ' ', 0)
	for _, f := range fields {
		fmt.Fprintf(tw, "%s\t%s\n", f.name, f.value)
	}
	return tw.Flush()
}

func (a *app) printTable(header []string, rows [][]string) error {
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	// Bold after alignment: tabwriter counts escape codes as width.
	head, rest, _ := strings.Cut(buf.String(), "\n")
	_, err := fmt.Fprintf(a.env.Stdout, "%s\n%s", a.color.bold(strings.TrimRight(head, " ")), rest)
	return err
}

func (a *app) info(format string, args ...any) {
	fmt.Fprintf(a.env.Stderr, format+"\n", args...)
}

func orDash[S ~string](s *S) string {
	if s == nil || *s == "" {
		return "-"
	}
	return string(*s)
}
