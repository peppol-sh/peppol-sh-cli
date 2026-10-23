package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

type harness struct {
	t      *testing.T
	env    Env
	vars   map[string]string
	stdout bytes.Buffer
	stderr bytes.Buffer
	api    *fakeAPI
	sleeps []time.Duration
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, vars: map[string]string{}, api: newFakeAPI(t)}
	h.env = Env{
		Stdin:     strings.NewReader(""),
		Stdout:    &h.stdout,
		Stderr:    &h.stderr,
		Getenv:    func(key string) string { return h.vars[key] },
		ConfigDir: filepath.Join(t.TempDir(), "peppol-sh"),
		Transport: h.api,
		Sleep: func(_ context.Context, d time.Duration) error {
			h.sleeps = append(h.sleeps, d)
			return nil
		},
	}
	return h
}

func (h *harness) run(args ...string) int {
	h.t.Helper()
	h.stdout.Reset()
	h.stderr.Reset()
	return Run(context.Background(), args, h.env)
}

func (h *harness) runWithStdin(stdin string, args ...string) int {
	h.t.Helper()
	h.env.Stdin = strings.NewReader(stdin)
	defer func() { h.env.Stdin = strings.NewReader("") }()
	return h.run(args...)
}

func (h *harness) wantExit(got, want int) {
	h.t.Helper()
	if got != want {
		h.t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", got, want, h.stdout.String(), h.stderr.String())
	}
}

func (h *harness) wantStderrContains(want string) {
	h.t.Helper()
	if !strings.Contains(h.stderr.String(), want) {
		h.t.Fatalf("stderr does not contain %q:\n%s", want, h.stderr.String())
	}
}

func (h *harness) wantStdoutContains(want string) {
	h.t.Helper()
	if !strings.Contains(h.stdout.String(), want) {
		h.t.Fatalf("stdout does not contain %q:\n%s", want, h.stdout.String())
	}
}

type reply struct {
	status int
	body   string
	header http.Header
}

func jsonReply(status int, body string) reply {
	return reply{status: status, body: body, header: http.Header{"Content-Type": {"application/json"}}}
}

// fakeAPI is the HTTP transport of the CLI under test. It records each request
// as the CLI sent it (so a test sees the host the CLI chose) and then reroutes
// it to a local httptest.Server, so no test touches the network.
type fakeAPI struct {
	t        *testing.T
	srv      *httptest.Server
	mux      *http.ServeMux
	mu       sync.Mutex
	requests []*http.Request
}

func newFakeAPI(t *testing.T) *fakeAPI {
	f := &fakeAPI{t: t, mux: http.NewServeMux()}
	f.srv = httptest.NewServer(f.mux)
	t.Cleanup(f.srv.Close)
	return f
}

// respond serves the replies in order for pattern; the last one repeats.
func (f *fakeAPI) respond(pattern string, replies ...reply) {
	var mu sync.Mutex
	n := 0
	f.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		rep := replies[min(n, len(replies)-1)]
		n++
		mu.Unlock()
		for k, v := range rep.header {
			w.Header()[k] = v
		}
		w.WriteHeader(rep.status)
		_, _ = io.WriteString(w, rep.body)
	})
}

func (f *fakeAPI) RoundTrip(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.requests = append(f.requests, req.Clone(context.Background()))
	f.mu.Unlock()

	out := req.Clone(req.Context())
	out.URL.Scheme = "http"
	out.URL.Host = strings.TrimPrefix(f.srv.URL, "http://")
	out.Host = ""
	return http.DefaultTransport.RoundTrip(out)
}

func (f *fakeAPI) sent() []*http.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

func (f *fakeAPI) count() int { return len(f.sent()) }

func (f *fakeAPI) last() *http.Request {
	f.t.Helper()
	sent := f.sent()
	if len(sent) == 0 {
		f.t.Fatal("the CLI sent no request")
	}
	return sent[len(sent)-1]
}
