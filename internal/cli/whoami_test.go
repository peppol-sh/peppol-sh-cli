package cli

import (
	"os"
	"path/filepath"
	"testing"
)

const account = `{"id":"acc_V1StGXR8Z5jdHi6B","email":"jane@example.com","name":"Jane Doe","status":"active",` +
	`"api_keys":[{"prefix":"ps_test_abcdefg...","sandbox":true,"label":null,"last_used_at":null,"created_at":"2026-03-05T12:00:00Z","revoked":false}],` +
	`"usage":{"total_documents_sent":3,"total_api_calls":42},"created_at":"2026-03-01T09:00:00Z"}`

const (
	flagKey   = "ps_test_flagflagflagF1a9"
	envKey    = "ps_test_envenvenvenvE2b8"
	configKey = "ps_live_configconfigC3c7"
)

func writeConfig(t *testing.T, h *harness, key string) {
	t.Helper()
	if err := os.MkdirAll(h.env.ConfigDir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := "api_key = \"" + key + "\"\n"
	if err := os.WriteFile(filepath.Join(h.env.ConfigDir, "config.toml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestKeyResolutionOrder(t *testing.T) {
	tests := []struct {
		name              string
		flag, env, config string
		wantAuthorization string
	}{
		{name: "flag beats env and config", flag: flagKey, env: envKey, config: configKey, wantAuthorization: "Bearer " + flagKey},
		{name: "env beats config", env: envKey, config: configKey, wantAuthorization: "Bearer " + envKey},
		{name: "config file last", config: configKey, wantAuthorization: "Bearer " + configKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			h.api.respond("GET /v1/account", jsonReply(200, account))
			if tt.env != "" {
				h.vars["PEPPOL_SH_API_KEY"] = tt.env
			}
			if tt.config != "" {
				writeConfig(t, h, tt.config)
			}
			args := []string{"whoami"}
			if tt.flag != "" {
				args = append(args, "--api-key", tt.flag)
			}

			h.wantExit(h.run(args...), 0)

			if got := h.api.last().Header.Get("Authorization"); got != tt.wantAuthorization {
				t.Fatalf("Authorization = %q, want %q", got, tt.wantAuthorization)
			}
		})
	}
}

func TestHostFollowsTheKeyPrefix(t *testing.T) {
	tests := []struct{ key, wantURL string }{
		{"ps_test_abcdefghijkl1234", "https://sandbox.peppol.sh/v1/account"},
		{"ps_live_abcdefghijkl1234", "https://api.peppol.sh/v1/account"},
	}
	for _, tt := range tests {
		t.Run(tt.key[:8], func(t *testing.T) {
			h := newHarness(t)
			h.api.respond("GET /v1/account", jsonReply(200, account))
			h.vars["PEPPOL_SH_API_KEY"] = tt.key

			h.wantExit(h.run("whoami"), 0)

			if got := h.api.last().URL.String(); got != tt.wantURL {
				t.Fatalf("request URL = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestBaseURLOverridesTheHost(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		env     string
		wantURL string
	}{
		{name: "env", env: "http://localhost:8787", wantURL: "http://localhost:8787/v1/account"},
		{name: "flag", flag: "http://127.0.0.1:9000/", wantURL: "http://127.0.0.1:9000/v1/account"},
		{name: "flag beats env", flag: "http://127.0.0.1:9000", env: "http://localhost:8787", wantURL: "http://127.0.0.1:9000/v1/account"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			h.api.respond("GET /v1/account", jsonReply(200, account))
			h.vars["PEPPOL_SH_API_KEY"] = "ps_live_abcdefghijkl1234"
			if tt.env != "" {
				h.vars["PEPPOL_SH_BASE_URL"] = tt.env
			}
			args := []string{"whoami"}
			if tt.flag != "" {
				args = append(args, "--base-url", tt.flag)
			}

			h.wantExit(h.run(args...), 0)

			if got := h.api.last().URL.String(); got != tt.wantURL {
				t.Fatalf("request URL = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestWhoamiPrintsTheAccountAndAMaskedKey(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/account", jsonReply(200, account))
	h.vars["PEPPOL_SH_API_KEY"] = "ps_test_abcdefghijkl1234"

	h.wantExit(h.run("whoami"), 0)

	want := "" +
		"Account      acc_V1StGXR8Z5jdHi6B\n" +
		"Email        jane@example.com\n" +
		"Name         Jane Doe\n" +
		"Status       active\n" +
		"Environment  sandbox\n" +
		"Host         https://sandbox.peppol.sh\n" +
		"API key      ps_test_...1234\n" +
		"Key source   PEPPOL_SH_API_KEY\n"
	if got := h.stdout.String(); got != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", got, want)
	}
	if h.stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want nothing", h.stderr.String())
	}
}

func TestWhoamiWithoutAKeyIsAnAuthError(t *testing.T) {
	h := newHarness(t)

	h.wantExit(h.run("whoami"), 3)

	h.wantStderrContains(`Run "peppol-sh login" or set PEPPOL_SH_API_KEY.`)
	if n := h.api.count(); n != 0 {
		t.Fatalf("sent %d requests without a key, want 0", n)
	}
}

func TestAKeyWithAnUnknownPrefixIsAnAuthError(t *testing.T) {
	h := newHarness(t)
	h.vars["PEPPOL_SH_API_KEY"] = "sk_live_abcdefghijkl1234"

	h.wantExit(h.run("whoami"), 3)

	h.wantStderrContains("ps_test_")
	if n := h.api.count(); n != 0 {
		t.Fatalf("sent %d requests with a bad key, want 0", n)
	}
}
