package cli

import (
	"runtime"
	"testing"
)

const healthOK = `{"status":"ok","version":"2.0.0","environment":"production","worker_version":null,"checks":{"db":"ok"}}`

const healthDegraded = `{"status":"degraded","version":"2.0.0","environment":"production","worker_version":null,"checks":{"db":"error"}}`

func TestHealthWithoutAKeyCallsTheLiveHost(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/health", jsonReply(200, healthOK))

	h.wantExit(h.run("health"), 0)

	if got := h.api.last().URL.String(); got != "https://api.peppol.sh/v1/health" {
		t.Fatalf("request URL = %q, want https://api.peppol.sh/v1/health", got)
	}
}

func TestHealthPrintsStatusVersionEnvironmentAndDatabase(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/health", jsonReply(200, healthOK))

	h.wantExit(h.run("health"), 0)

	want := "" +
		"Status       ok\n" +
		"API version  2.0.0\n" +
		"Environment  production\n" +
		"Database     ok\n"
	if got := h.stdout.String(); got != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", got, want)
	}
}

func TestHealthDegradedPrintsTheChecksAndExits1(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/health", jsonReply(503, healthDegraded))

	h.wantExit(h.run("health"), 1)

	h.wantStdoutContains("Status       degraded\n")
	h.wantStdoutContains("Database     error\n")
	h.wantStderrContains("degraded")
}

func TestHealthDoesNotSendTheKey(t *testing.T) {
	h := newHarness(t)
	h.vars["PEPPOL_SH_API_KEY"] = "ps_test_abcdefghijkl1234"
	h.api.respond("GET /v1/health", jsonReply(200, healthOK))

	h.wantExit(h.run("health"), 0)

	if got := h.api.last().Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want none", got)
	}
}

func TestHealthIgnoresAKeyWithAnUnknownPrefix(t *testing.T) {
	h := newHarness(t)
	h.vars["PEPPOL_SH_API_KEY"] = "junk"
	h.api.respond("GET /v1/health", jsonReply(200, healthOK))

	h.wantExit(h.run("health"), 0)

	req := h.api.last()
	if got := req.URL.String(); got != "https://api.peppol.sh/v1/health" {
		t.Fatalf("request URL = %q, want https://api.peppol.sh/v1/health", got)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want none", got)
	}
	want := "Warning: ignoring the API key from PEPPOL_SH_API_KEY: it does not start with ps_test_ or ps_live_.\n"
	if got := h.stderr.String(); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestEveryRequestCarriesTheUserAgent(t *testing.T) {
	setVersion(t, "9.8.7")
	h := newHarness(t)
	h.api.respond("GET /v1/health", jsonReply(200, healthOK))

	h.wantExit(h.run("health"), 0)

	want := "peppol-sh/9.8.7 (" + runtime.GOOS + "/" + runtime.GOARCH + ")"
	if got := h.api.last().Header.Get("User-Agent"); got != want {
		t.Fatalf("User-Agent = %q, want %q", got, want)
	}
}
