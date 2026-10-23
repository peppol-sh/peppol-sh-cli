package cli

import (
	"context"
	"net/http"

	"github.com/peppol-sh/peppol-sh-cli/internal/api"
	"github.com/spf13/cobra"
)

func newHealthCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:         "health",
		Short:       "Check the status of the API",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{"operationId": "getHealth"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			send := func(ctx context.Context, c *api.Client) (*http.Response, error) { return c.GetHealth(ctx) }
			status, err := request(cmd, a, keyHostOnly, send, a.printHealth, http.StatusOK, http.StatusServiceUnavailable)
			if err != nil {
				return err
			}
			if status == http.StatusServiceUnavailable {
				return newError(exitError, "the API is degraded (HTTP 503)")
			}
			return nil
		},
	}
}

func (a *app) printHealth(_ *session, h *api.HealthResponse) error {
	return a.printFields(
		field{"Status", a.color.state(string(h.Status), h.Status == api.HealthResponseStatusOk)},
		field{"API version", h.Version},
		field{"Environment", string(h.Environment)},
		field{"Database", a.color.state(string(h.Checks.Db), h.Checks.Db == api.HealthResponseChecksDbOk)},
	)
}
