package api_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/peppol-sh/peppol-sh-cli/internal/spectest"
	"gopkg.in/yaml.v3"
)

func TestGeneratedClientMatchesTheSpec(t *testing.T) {
	specPath := spectest.Path(t)

	dir := t.TempDir()
	out := filepath.Join(dir, "client.gen.go")
	config := filepath.Join(dir, "oapi-codegen.yaml")
	writeConfigWithOutput(t, "oapi-codegen.yaml", config, out)

	cmd := exec.Command("go", "tool", "oapi-codegen", "-config", config, specPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("oapi-codegen: %v\n%s", err, output)
	}

	if !bytes.Equal(readLF(t, "client.gen.go"), readLF(t, out)) {
		t.Fatal("client.gen.go is out of date with openapi.yaml: run `go generate ./...` in packages/cli")
	}
}

// oapi-codegen ignores -o when the config sets output, so the test writes a
// copy of the config that points output at a temp file.
func writeConfigWithOutput(t *testing.T, src, dst, output string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := yaml.Unmarshal(b, &config); err != nil {
		t.Fatal(err)
	}
	config["output"] = output
	b, err = yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readLF(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}
