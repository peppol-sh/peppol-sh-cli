// Package spectest finds the OpenAPI spec for the tests that need it.
package spectest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Path returns the path of apps/api/src/openapi.yaml. The spec exists only in
// the monorepo, so Path skips t without it, or fails t when
// PEPPOL_SH_REQUIRE_DRIFT_CHECK=1.
func Path(t testing.TB) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "apps", "api", "src", "openapi.yaml")
	if _, err := os.Stat(path); err != nil {
		if os.Getenv("PEPPOL_SH_REQUIRE_DRIFT_CHECK") == "1" {
			t.Fatalf("PEPPOL_SH_REQUIRE_DRIFT_CHECK=1 but the spec is missing: %v", err)
		}
		t.Skipf("no spec at %s", path)
	}
	return path
}
