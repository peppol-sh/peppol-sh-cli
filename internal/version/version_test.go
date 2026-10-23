package version_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/peppol-sh/peppol-sh-cli/internal/version"
)

// The mirror tags the release from the newest CHANGELOG heading, so the
// constant must not drift from it.
func TestVersionMatchesTheNewestChangelogRelease(t *testing.T) {
	changelog, err := os.ReadFile(filepath.Join("..", "..", "CHANGELOG.md"))
	if os.IsNotExist(err) {
		t.Skip("no CHANGELOG.md in this checkout")
	}
	if err != nil {
		t.Fatal(err)
	}

	heading := regexp.MustCompile(`(?m)^##\s*\[?v?(\d+\.\d+\.\d+[^\]\s]*)\]?`)
	match := heading.FindSubmatch(changelog)
	if match == nil {
		t.Fatal("CHANGELOG.md has no ## [X.Y.Z] heading")
	}
	if got, want := version.Version, string(match[1]); got != want {
		t.Fatalf("version.Version = %q, newest CHANGELOG.md release is %q", got, want)
	}
}
