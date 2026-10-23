package cli

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/peppol-sh/peppol-sh-cli/internal/version"
	"github.com/spf13/cobra"
)

// Run executes the CLI with args (without the program name) and returns the
// exit code.
func Run(ctx context.Context, args []string, env Env) int {
	a := &app{env: env, color: palette{on: isTerminal(env.Stdout) && env.Getenv("NO_COLOR") == ""}}
	root := newRootCmd(a)
	root.SetArgs(args)
	cmd, err := root.ExecuteContextC(ctx)
	if err == nil {
		return exitOK
	}
	hint := fmt.Sprintf("Run %q for usage.", cmd.CommandPath()+" --help")
	return render(env.Stderr, err, hint)
}

// app is the state of one run, shared by the commands.
type app struct {
	env     Env
	apiKey  string
	baseURL string
	json    bool
	color   palette
}

func platform() string { return runtime.GOOS + "/" + runtime.GOARCH }

func newRootCmd(a *app) *cobra.Command {
	root := &cobra.Command{
		Use:   "peppol-sh",
		Short: "Send Peppol e-invoices with the peppol.sh API",
		Long: "peppol-sh is the command-line client of the peppol.sh API.\n\n" +
			"Environment:\n" +
			"  PEPPOL_SH_API_KEY   API key: ps_test_... (sandbox) or ps_live_... (live)\n" +
			"  PEPPOL_SH_BASE_URL  API base URL, instead of the host for the key\n" +
			"  NO_COLOR            no colour in the output\n\n" +
			"Exit codes: 0 OK, 1 API or runtime error, 2 usage error, 3 auth error.",
		Version:       version.Version + " (" + platform() + ")",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetVersionTemplate("peppol-sh {{.Version}}\n")
	root.SetIn(a.env.Stdin)
	root.SetOut(a.env.Stdout)
	root.SetErr(a.env.Stderr)

	flags := root.PersistentFlags()
	flags.StringVar(&a.apiKey, "api-key", "", "API key (default: $PEPPOL_SH_API_KEY, then the stored key); other processes can see a flag, so $PEPPOL_SH_API_KEY or login is safer")
	flags.StringVar(&a.baseURL, "base-url", "", "API base URL (default: $PEPPOL_SH_BASE_URL, then from the key)")
	flags.BoolVar(&a.json, "json", false, "print the raw API response as JSON")

	root.AddCommand(
		newVersionCmd(a),
		newHealthCmd(a),
		newLoginCmd(a),
		newLogoutCmd(a),
		newWhoamiCmd(a),
		newCompaniesCmd(a),
		newLookupCmd(a),
	)
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	asRuntimeErrors(root)
	return root
}

// asRuntimeErrors makes a plain error from a command (a failed write, say) exit
// with 1, so that render can treat every other plain error as cobra's usage error.
func asRuntimeErrors(c *cobra.Command) {
	if run := c.RunE; run != nil {
		c.RunE = func(cmd *cobra.Command, args []string) error {
			err := run(cmd, args)
			var ce *cliError
			if err != nil && !errors.As(err, &ce) {
				return newError(exitError, "%s", err)
			}
			return err
		}
	}
	for _, sub := range c.Commands() {
		asRuntimeErrors(sub)
	}
}
