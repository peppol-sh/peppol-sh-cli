package cli

import (
	"context"
	"net/http"
	"time"

	"github.com/peppol-sh/peppol-sh-cli/internal/api"
	"github.com/spf13/cobra"
)

func newCompaniesCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "companies",
		Short: "Work with the companies of your workspace",
		// A RunE with NoArgs makes "companies <typo>" a usage error; without it
		// cobra prints the help and exits 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(newCompaniesListCmd(a))
	return cmd
}

// companyList is the 200 body of listCompanies, an inline schema in the spec.
type companyList struct {
	Data *[]api.CompanyListItem `json:"data,omitempty"`
}

func newCompaniesListCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:         "list",
		Short:       "List the companies of your workspace",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{"operationId": "listCompanies"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			send := func(ctx context.Context, c *api.Client) (*http.Response, error) { return c.ListCompanies(ctx) }
			_, err := request(cmd, a, keyRequired, send, a.printCompanies)
			return err
		},
	}
}

func (a *app) printCompanies(_ *session, list *companyList) error {
	if list.Data == nil || len(*list.Data) == 0 {
		a.info("No companies in this workspace.")
		return nil
	}
	var rows [][]string
	for _, c := range *list.Data {
		live := "no"
		if c.IsLive != nil && *c.IsLive {
			live = "yes"
		}
		created := "-"
		if c.CreatedAt != nil {
			created = c.CreatedAt.UTC().Format(time.DateOnly)
		}
		rows = append(rows, []string{orDash(c.Id), orDash(c.Name), orDash(c.Country), orDash(c.TaxId), live, created})
	}
	return a.printTable([]string{"ID", "NAME", "COUNTRY", "TAX ID", "LIVE", "CREATED"}, rows)
}
