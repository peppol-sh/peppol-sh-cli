package cli

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
)

const rateLimited = `{"error":{"type":"rate_limit_error","code":"rate_limit_exceeded","message":"Too many requests"}}`

func rateLimitReply(retryAfter string) reply {
	return reply{status: 429, body: rateLimited, header: http.Header{
		"Content-Type": {"application/json"},
		"Retry-After":  {retryAfter},
	}}
}

func TestA429IsRetriedOnceAfterRetryAfter(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/health", rateLimitReply("2"), jsonReply(200, healthOK))

	h.wantExit(h.run("health"), 0)

	if want := []time.Duration{2 * time.Second}; !slices.Equal(h.sleeps, want) {
		t.Fatalf("slept %v, want %v", h.sleeps, want)
	}
	if n := h.api.count(); n != 2 {
		t.Fatalf("sent %d requests, want 2", n)
	}
	h.wantStdoutContains("Status       ok\n")
}

func TestA429AfterTheRetryFailsAndSaysHowLongToWait(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/health", rateLimitReply("2"), rateLimitReply("30"))

	h.wantExit(h.run("health"), 1)

	if want := []time.Duration{2 * time.Second}; !slices.Equal(h.sleeps, want) {
		t.Fatalf("slept %v, want %v", h.sleeps, want)
	}
	h.wantStderrContains("Error: rate_limit_exceeded: Too many requests (HTTP 429)\n")
	h.wantStderrContains("Wait 30 seconds, then try again.")
}

func TestAWaitOfOneSecondIsSingular(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/health", rateLimitReply("1"))

	h.wantExit(h.run("health"), 1)

	h.wantStderrContains("Rate limit reached. Wait 1 second, then try again.\n")
}

func TestCancellingTheRunStopsTheRetryWait(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/health", rateLimitReply("60"))
	h.env.Sleep = sleep
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.env.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		resp, err := h.api.RoundTrip(r)
		cancel()
		return resp, err
	})

	done := make(chan int, 1)
	go func() { done <- Run(ctx, []string{"health"}, h.env) }()
	select {
	case code := <-done:
		h.wantExit(code, 1)
	case <-time.After(10 * time.Second):
		t.Fatal("the run still waits for Retry-After after its context was cancelled")
	}
	if n := h.api.count(); n != 1 {
		t.Fatalf("sent %d requests, want 1", n)
	}
}

func TestARetryAfterOverTheCapFailsWithoutWaiting(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/health", rateLimitReply("120"), jsonReply(200, healthOK))

	h.wantExit(h.run("health"), 1)

	if len(h.sleeps) != 0 {
		t.Fatalf("slept %v, want no sleep", h.sleeps)
	}
	if n := h.api.count(); n != 1 {
		t.Fatalf("sent %d requests, want 1", n)
	}
	h.wantStderrContains("Wait 120 seconds, then try again.")
}
