package cli

import (
	"context"
	"net/http"

	"github.com/peppol-sh/peppol-sh-cli/internal/api"
	"github.com/spf13/cobra"
)

func newWhoamiCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:         "whoami",
		Short:       "Show the account of the API key",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{"operationId": "getAccount"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := request(cmd, a, keyRequired, getAccount, a.printAccount)
			return err
		},
	}
}

func getAccount(ctx context.Context, c *api.Client) (*http.Response, error) {
	return c.GetAccount(ctx)
}

func (a *app) printAccount(s *session, acc *api.Account) error {
	return a.printFields(
		field{"Account", orDash(acc.Id)},
		field{"Email", orDash(acc.Email)},
		field{"Name", orDash(acc.Name)},
		field{"Status", orDash(acc.Status)},
		field{"Environment", s.cred.environment()},
		field{"Host", s.host},
		field{"API key", s.cred.masked()},
		field{"Key source", s.cred.source},
	)
}
