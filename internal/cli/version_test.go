package cli

import (
	"runtime"
	"testing"

	"github.com/peppol-sh/peppol-sh-cli/internal/version"
)

// setVersion stands in for the -ldflags -X of a release build.
func setVersion(t *testing.T, v string) {
	t.Helper()
	old := version.Version
	version.Version = v
	t.Cleanup(func() { version.Version = old })
}

func TestVersionPrintsNameVersionAndPlatform(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}} {
		t.Run(args[0], func(t *testing.T) {
			setVersion(t, "9.8.7")
			h := newHarness(t)

			h.wantExit(h.run(args...), 0)

			want := "peppol-sh 9.8.7 (" + runtime.GOOS + "/" + runtime.GOARCH + ")\n"
			if got := h.stdout.String(); got != want {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
		})
	}
}

func TestVersionJSON(t *testing.T) {
	setVersion(t, "9.8.7")
	h := newHarness(t)

	h.wantExit(h.run("version", "--json"), 0)

	want := `{"version":"9.8.7","os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `"}` + "\n"
	if got := h.stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}
