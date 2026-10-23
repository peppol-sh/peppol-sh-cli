package cli

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/peppol-sh/peppol-sh-cli/internal/version"
)

const maxRetryWait = 60 * time.Second

func userAgent() string { return "peppol-sh/" + version.Version + " (" + platform() + ")" }

func (a *app) transport() http.RoundTripper {
	return &transport{base: a.env.Transport, sleep: a.env.Sleep}
}

// transport sets the User-Agent and retries a 429 once after Retry-After.
type transport struct {
	base  http.RoundTripper
	sleep func(context.Context, time.Duration) error
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	first := req.Clone(req.Context())
	first.Header.Set("User-Agent", userAgent())
	resp, err := t.base.RoundTrip(first)
	if err != nil || resp.StatusCode != http.StatusTooManyRequests {
		return resp, err
	}
	wait, ok := retryAfter(resp.Header)
	// The first try has consumed a request body, so a request with one is not retried.
	if !ok || wait > maxRetryWait || req.Body != nil {
		return resp, nil
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	if err := t.sleep(req.Context(), wait); err != nil {
		return nil, err
	}
	return t.base.RoundTrip(first.Clone(req.Context()))
}

func retryAfter(h http.Header) (time.Duration, bool) {
	seconds, err := strconv.Atoi(strings.TrimSpace(h.Get("Retry-After")))
	if err != nil || seconds < 0 {
		return 0, false
	}
	return time.Duration(seconds) * time.Second, true
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
