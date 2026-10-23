package cli

import (
	"context"
	"fmt"
	"net/http"

	"github.com/peppol-sh/peppol-sh-cli/internal/api"
	"github.com/spf13/cobra"
)

func newLookupCmd(a *app) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "lookup <peppol_id>",
		Short: "Look up a Peppol participant and the documents it can receive",
		Example: "  peppol-sh lookup 0208:0123456749\n" +
			"  peppol-sh lookup 0208:0123456749 --domain smk",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{"operationId": "lookupParticipant"},
		RunE: func(cmd *cobra.Command, args []string) error {
			d := api.LookupParticipantParamsDomain(domain)
			if !d.Valid() {
				return newError(exitUsage, "--domain must be sml or smk, got %q", domain).
					with(`Run "peppol-sh lookup --help" for usage.`)
			}
			params := &api.LookupParticipantParams{}
			if cmd.Flags().Changed("domain") {
				params.Domain = &d
			}
			send := func(ctx context.Context, c *api.Client) (*http.Response, error) {
				return c.LookupParticipant(ctx, args[0], params)
			}
			_, err := request(cmd, a, keyOptional, send, a.printLookup)
			return err
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "sml", "SML domain: sml (production) or smk (test)")
	return cmd
}

func (a *app) printLookup(_ *session, r *api.SmpLookupResult) error {
	participant := "-"
	if p := r.ParticipantId; p != nil {
		participant = orDash(p.Scheme) + ":" + orDash(p.Id)
	}
	fields := []field{
		{"Participant", participant},
		{"Domain", orDash(r.Domain)},
		{"SMP", orDash(r.SmpUrl)},
	}
	if bc := r.BusinessCard; bc != nil && bc.Entities != nil {
		for _, e := range *bc.Entities {
			entity := orDash(e.Name)
			if e.CountryCode != nil && *e.CountryCode != "" {
				entity += fmt.Sprintf(" (%s)", *e.CountryCode)
			}
			fields = append(fields, field{"Entity", entity})
		}
	}
	if err := a.printFields(fields...); err != nil {
		return err
	}

	if r.Services == nil || len(*r.Services) == 0 {
		a.info("The participant lists no services.")
		return nil
	}
	var rows [][]string
	for _, svc := range *r.Services {
		rows = append(rows, []string{
			firstOf(svc.ProcessName, svc.ProcessId),
			firstOf(svc.DocumentTypeName, svc.DocumentTypeId),
		})
	}
	if _, err := fmt.Fprintln(a.env.Stdout); err != nil {
		return err
	}
	// The document type is last: a raw document type ID is long, and the last
	// column is not padded.
	return a.printTable([]string{"PROCESS", "DOCUMENT TYPE"}, rows)
}

func firstOf(name, id *string) string {
	if name != nil && *name != "" {
		return *name
	}
	return orDash(id)
}
