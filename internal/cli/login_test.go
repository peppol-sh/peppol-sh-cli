package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func (h *harness) configFile() string { return filepath.Join(h.env.ConfigDir, "config.toml") }

func (h *harness) wantNoConfigFile() {
	h.t.Helper()
	if _, err := os.Stat(h.configFile()); !os.IsNotExist(err) {
		h.t.Fatalf("config file exists (err = %v), want none", err)
	}
}

func TestLoginVerifiesTheKeyAgainstItsHostAndStoresIt(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/account", jsonReply(200, account))

	h.wantExit(h.runWithStdin("ps_test_abcdefghijkl1234\r\n", "login"), 0)

	req := h.api.last()
	if got := req.URL.String(); got != "https://sandbox.peppol.sh/v1/account" {
		t.Fatalf("request URL = %q, want https://sandbox.peppol.sh/v1/account", got)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer ps_test_abcdefghijkl1234" {
		t.Fatalf("Authorization = %q", got)
	}
	if got, want := h.stderr.String(), "Logged in as jane@example.com (sandbox, key ps_test_...1234).\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestTheStoredKeyIsUsedByWhoami(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/account", jsonReply(200, account))
	h.wantExit(h.runWithStdin("ps_live_abcdefghijkl5678\n", "login"), 0)

	h.wantExit(h.run("whoami"), 0)

	if got := h.api.last().Header.Get("Authorization"); got != "Bearer ps_live_abcdefghijkl5678" {
		t.Fatalf("Authorization = %q, want the stored key", got)
	}
	h.wantStdoutContains("Key source   config file " + h.configFile() + "\n")
}

func TestLoginWritesTheConfigOnlyForTheUser(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no Unix file modes")
	}
	h := newHarness(t)
	h.api.respond("GET /v1/account", jsonReply(200, account))

	h.wantExit(h.runWithStdin("ps_test_abcdefghijkl1234\n", "login"), 0)

	file, err := os.Stat(h.configFile())
	if err != nil {
		t.Fatal(err)
	}
	if got := file.Mode().Perm(); got != 0o600 {
		t.Fatalf("config.toml mode = %o, want 600", got)
	}
	dir, err := os.Stat(h.env.ConfigDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := dir.Mode().Perm(); got != 0o700 {
		t.Fatalf("config dir mode = %o, want 700", got)
	}
}

func TestLoginAcceptsTheKeyFlag(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/account", jsonReply(200, account))

	h.wantExit(h.run("login", "--api-key", "ps_live_abcdefghijkl5678"), 0)

	if got := h.api.last().URL.String(); got != "https://api.peppol.sh/v1/account" {
		t.Fatalf("request URL = %q, want https://api.peppol.sh/v1/account", got)
	}
	h.wantStderrContains("(live, key ps_live_...5678)")
}

func TestLoginHonoursTheBaseURLEnv(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/account", jsonReply(200, account))
	h.vars["PEPPOL_SH_BASE_URL"] = "http://localhost:8787"

	h.wantExit(h.runWithStdin("ps_test_abcdefghijkl1234\n", "login"), 0)

	if got := h.api.last().URL.String(); got != "http://localhost:8787/v1/account" {
		t.Fatalf("request URL = %q, want http://localhost:8787/v1/account", got)
	}
}

func TestLoginWithARejectedKeyWritesNothing(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/account", jsonReply(401,
		`{"error":{"type":"authentication_error","code":"invalid_api_key","message":"Invalid API key"}}`))

	h.wantExit(h.runWithStdin("ps_test_abcdefghijkl1234\n", "login"), 3)

	h.wantStderrContains("invalid_api_key")
	h.wantNoConfigFile()
}

func TestLoginWithABadPrefixWritesNothing(t *testing.T) {
	h := newHarness(t)

	h.wantExit(h.runWithStdin("sk_live_abcdefghijkl1234\n", "login"), 3)

	if n := h.api.count(); n != 0 {
		t.Fatalf("sent %d requests, want 0", n)
	}
	h.wantNoConfigFile()
}

func TestLoginWithoutAKeyIsAnAuthError(t *testing.T) {
	h := newHarness(t)

	h.wantExit(h.runWithStdin("\n", "login"), 3)

	h.wantNoConfigFile()
}

func TestLogoutRemovesTheStoredKey(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/account", jsonReply(200, account))
	h.wantExit(h.runWithStdin("ps_test_abcdefghijkl1234\n", "login"), 0)

	h.wantExit(h.run("logout"), 0)

	h.wantNoConfigFile()
	h.wantExit(h.run("whoami"), 3)
}

func TestLogoutWithNothingStoredSaysSo(t *testing.T) {
	h := newHarness(t)

	h.wantExit(h.run("logout"), 0)

	h.wantStderrContains("No API key is stored.")
}
