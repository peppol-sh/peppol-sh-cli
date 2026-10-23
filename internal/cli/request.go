package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/peppol-sh/peppol-sh-cli/internal/api"
	"github.com/spf13/cobra"
)

type sendFunc func(context.Context, *api.Client) (*http.Response, error)

type response[T any] struct {
	status int
	body   []byte
	data   *T
}

// request opens a session, sends one request, and prints the response: the
// body unchanged with --json, else through print. It returns the status, one
// of ok (200 when ok is empty).
func request[T any](cmd *cobra.Command, a *app, use keyUse, send sendFunc, print func(*session, *T) error, ok ...int) (int, error) {
	s, err := a.open(use)
	if err != nil {
		return 0, err
	}
	r, err := fetch[T](cmd.Context(), s, send, ok...)
	if err != nil {
		return 0, err
	}
	if a.json {
		return r.status, a.writeRaw(r.body)
	}
	return r.status, print(s, r.data)
}

// fetch sends one request and decodes the body as T. A status not in ok (200
// when ok is empty) is an API error. A body that is not JSON, such as an HTML
// page from a proxy, is an unexpected response.
func fetch[T any](ctx context.Context, s *session, send sendFunc, ok ...int) (*response[T], error) {
	resp, err := send(ctx, s.client)
	if err != nil {
		return nil, newError(exitError, "request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newError(exitError, "request failed: %v", err)
	}
	if len(ok) == 0 {
		ok = []int{http.StatusOK}
	}
	if !slices.Contains(ok, resp.StatusCode) {
		return nil, apiError(resp, body)
	}
	var data T
	if !strings.Contains(resp.Header.Get("Content-Type"), "json") || json.Unmarshal(body, &data) != nil {
		return nil, newError(exitError, "unexpected response from the API (HTTP %d, Content-Type %q)",
			resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	return &response[T]{status: resp.StatusCode, body: body, data: &data}, nil
}
