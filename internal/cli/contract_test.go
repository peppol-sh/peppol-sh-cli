package cli

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/peppol-sh/peppol-sh-cli/internal/spectest"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Commands that call no API. A command here covers the commands below it.
var noAPICommands = []string{
	"peppol-sh version",
	"peppol-sh logout",
	"peppol-sh completion",
	"peppol-sh help",
}

// Args that run each annotated command, by command path.
var contractArgs = map[string][]string{
	"peppol-sh health":         {"health"},
	"peppol-sh login":          {"login", "--api-key", sandboxKey},
	"peppol-sh whoami":         {"whoami"},
	"peppol-sh companies list": {"companies", "list"},
	"peppol-sh lookup":         {"lookup", "0208:0123456749", "--domain", "smk"},
}

type operation struct{ method, path string }

func loadOperations(t *testing.T) map[string]operation {
	t.Helper()
	b, err := os.ReadFile(spectest.Path(t))
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal(b, &spec); err != nil {
		t.Fatal(err)
	}
	ops := map[string]operation{}
	for path, item := range spec.Paths {
		for method, node := range item {
			var op struct {
				OperationID string `yaml:"operationId"`
			}
			if node.Kind != yaml.MappingNode || node.Decode(&op) != nil || op.OperationID == "" {
				continue
			}
			ops[op.OperationID] = operation{method: strings.ToUpper(method), path: path}
		}
	}
	return ops
}

func walk(c *cobra.Command, visit func(*cobra.Command)) {
	visit(c)
	for _, sub := range c.Commands() {
		walk(sub, visit)
	}
}

// leafCommands are the runnable commands that are not a group of others.
func leafCommands() []*cobra.Command {
	var leaves []*cobra.Command
	walk(newRootCmd(&app{env: Env{Getenv: func(string) string { return "" }}}), func(c *cobra.Command) {
		if c.Runnable() && !c.HasSubCommands() {
			leaves = append(leaves, c)
		}
	})
	return leaves
}

func callsNoAPI(path string) bool {
	for _, p := range noAPICommands {
		if path == p || strings.HasPrefix(path, p+" ") {
			return true
		}
	}
	return false
}

func matchesTemplate(path, template string) bool {
	got, want := strings.Split(path, "/"), strings.Split(template, "/")
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		isParam := strings.HasPrefix(want[i], "{") && strings.HasSuffix(want[i], "}")
		if isParam && got[i] == "" || !isParam && got[i] != want[i] {
			return false
		}
	}
	return true
}

// Runs without the spec too, so that the mirror still catches a new command
// that nobody classified.
func TestEveryCommandIsAnnotatedOrCallsNoAPI(t *testing.T) {
	for _, c := range leafCommands() {
		path := c.CommandPath()
		if _, annotated := c.Annotations["operationId"]; !annotated {
			if !callsNoAPI(path) {
				t.Errorf("%q has no operationId annotation and is not in noAPICommands", path)
			}
			continue
		}
		if _, ok := contractArgs[path]; !ok {
			t.Errorf("contractArgs has no entry for %q", path)
		}
	}
}

func TestEveryCommandCallsTheOperationItIsAnnotatedWith(t *testing.T) {
	ops := loadOperations(t)

	for _, c := range leafCommands() {
		path := c.CommandPath()
		opID, annotated := c.Annotations["operationId"]
		args, hasArgs := contractArgs[path]
		if !annotated || !hasArgs {
			continue
		}
		t.Run(path, func(t *testing.T) {
			op, ok := ops[opID]
			if !ok {
				t.Fatalf("operationId %q is not in openapi.yaml", opID)
			}

			h := newHarness(t)
			h.vars["PEPPOL_SH_API_KEY"] = sandboxKey
			h.api.respond("/", jsonReply(200, `{}`))
			h.run(args...)

			sent := h.api.sent()
			if len(sent) == 0 {
				t.Fatalf("%q sent no request, want %s %s", path, op.method, op.path)
			}
			for _, req := range sent {
				if req.Method != op.method || !matchesTemplate(req.URL.Path, op.path) {
					t.Errorf("%q sent %s %s, want %s %s (%s)", path, req.Method, req.URL.Path, op.method, op.path, opID)
				}
			}
		})
	}
}

func TestCommandsWithoutAnOperationCallNoAPI(t *testing.T) {
	for _, c := range leafCommands() {
		path := c.CommandPath()
		if _, annotated := c.Annotations["operationId"]; annotated || !callsNoAPI(path) {
			continue
		}
		t.Run(path, func(t *testing.T) {
			h := newHarness(t)
			h.vars["PEPPOL_SH_API_KEY"] = sandboxKey
			h.api.respond("/", reply{status: http.StatusTeapot})

			h.wantExit(h.run(strings.Fields(path)[1:]...), 0)

			if n := h.api.count(); n != 0 {
				t.Fatalf("%q sent %d requests, want 0", path, n)
			}
		})
	}
}
