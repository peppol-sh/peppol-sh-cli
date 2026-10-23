package cli

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"testing"
)

const sandboxKey = "ps_test_abcdefghijkl1234"

func TestAPIErrorsShowCodeMessageAndStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantExit   int
		wantStderr string
	}{
		{
			name:       "401 is an auth error",
			status:     401,
			body:       `{"error":{"type":"authentication_error","code":"invalid_api_key","message":"Invalid API key"}}`,
			wantExit:   3,
			wantStderr: "Error: invalid_api_key: Invalid API key (HTTP 401)\n",
		},
		{
			name:       "403 is an auth error",
			status:     403,
			body:       `{"error":{"type":"authentication_error","code":"disabled_account","message":"Account is disabled"}}`,
			wantExit:   3,
			wantStderr: "Error: disabled_account: Account is disabled (HTTP 403)\n",
		},
		{
			name:       "500 shows the request ID",
			status:     500,
			body:       `{"error":{"type":"internal_error","code":"internal_error","message":"Internal server error","request_id":"8f2c1a3b4d5e6f70-AMS"}}`,
			wantExit:   1,
			wantStderr: "Error: internal_error: Internal server error (HTTP 500)\nRequest ID: 8f2c1a3b4d5e6f70-AMS\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			h.api.respond("GET /v1/account", jsonReply(tt.status, tt.body))
			h.vars["PEPPOL_SH_API_KEY"] = sandboxKey

			h.wantExit(h.run("whoami"), tt.wantExit)

			if got := h.stderr.String(); got != tt.wantStderr {
				t.Fatalf("stderr = %q, want %q", got, tt.wantStderr)
			}
			if h.stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want nothing", h.stdout.String())
			}
		})
	}
}

func TestAnErrorWithoutAnErrorObjectShowsTheStatus(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/account", reply{status: 502, body: "<html>Bad gateway</html>", header: http.Header{"Content-Type": {"text/html"}}})
	h.vars["PEPPOL_SH_API_KEY"] = sandboxKey

	h.wantExit(h.run("whoami"), 1)

	if got, want := h.stderr.String(), "Error: HTTP 502 Bad Gateway\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestANetworkErrorExits1(t *testing.T) {
	h := newHarness(t)
	h.env.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial tcp: lookup api.peppol.sh: no such host")
	})

	h.wantExit(h.run("health"), 1)

	h.wantStderrContains("no such host")
}

func TestUsageErrorsExit2WithAHelpHint(t *testing.T) {
	tests := []struct {
		name string
		args []string
		hint string
	}{
		{name: "unknown flag", args: []string{"whoami", "--nope"}, hint: `Run "peppol-sh whoami --help" for usage.`},
		{name: "wrong arg count", args: []string{"version", "extra"}, hint: `Run "peppol-sh version --help" for usage.`},
		{name: "unknown command", args: []string{"frobnicate"}, hint: `Run "peppol-sh --help" for usage.`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)

			h.wantExit(h.run(tt.args...), 2)

			h.wantStderrContains("Error: ")
			h.wantStderrContains(tt.hint)
		})
	}
}

func TestJSONPrintsTheResponseBodyUnchanged(t *testing.T) {
	const body = `{"id":"acc_V1StGXR8Z5jdHi6B",  "email":"jane@example.com" ,"name":null,"status":"active","future_field":[1,2]}`
	h := newHarness(t)
	h.api.respond("GET /v1/account", jsonReply(200, body))
	h.vars["PEPPOL_SH_API_KEY"] = sandboxKey

	h.wantExit(h.run("whoami", "--json"), 0)

	if !bytes.Equal(h.stdout.Bytes(), []byte(body)) {
		t.Fatalf("stdout = %q, want the body %q", h.stdout.String(), body)
	}
	if strings.TrimSpace(h.stderr.String()) != "" {
		t.Fatalf("stderr = %q, want nothing", h.stderr.String())
	}
}

func TestJSONNeverPrintsAnHTMLPage(t *testing.T) {
	tests := []struct {
		pattern string
		args    []string
	}{
		{"GET /v1/account", []string{"whoami", "--json"}},
		{"GET /v1/companies", []string{"companies", "list", "--json"}},
		{"GET /v1/lookup/{peppol_id}", []string{"lookup", "0208:0123456749", "--json"}},
		{"GET /v1/health", []string{"health", "--json"}},
	}
	for _, tt := range tests {
		t.Run(tt.args[0], func(t *testing.T) {
			h := newHarness(t)
			h.api.respond(tt.pattern, reply{status: 200, body: "<html>Sign in to the proxy</html>", header: http.Header{"Content-Type": {"text/html"}}})
			h.vars["PEPPOL_SH_API_KEY"] = sandboxKey

			h.wantExit(h.run(tt.args...), 1)

			if h.stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want nothing", h.stdout.String())
			}
			h.wantStderrContains(`Error: unexpected response from the API (HTTP 200, Content-Type "text/html")`)
		})
	}
}
