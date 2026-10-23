package cli

import (
	"fmt"
	"runtime"

	"github.com/peppol-sh/peppol-sh-cli/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of peppol-sh",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			if a.json {
				return a.writeJSON(struct {
					Version string `json:"version"`
					OS      string `json:"os"`
					Arch    string `json:"arch"`
				}{version.Version, runtime.GOOS, runtime.GOARCH})
			}
			_, err := fmt.Fprintf(a.env.Stdout, "peppol-sh %s (%s)\n", version.Version, platform())
			return err
		},
	}
}
