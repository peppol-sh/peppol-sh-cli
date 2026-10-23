package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/peppol-sh/peppol-sh-cli/internal/api"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newLoginCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Verify an API key and store it for later commands",
		Long: "Verify an API key with the API and store it in the config file.\n\n" +
			"The key is read from a hidden prompt, or from the first line of stdin:\n\n" +
			"  echo \"$PEPPOL_SH_API_KEY\" | peppol-sh login",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{"operationId": "getAccount"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			key := a.apiKey
			if key == "" {
				var err error
				if key, err = a.readKey(); err != nil {
					return newError(exitError, "cannot read the API key: %v", err)
				}
			}
			if key == "" {
				return newError(exitAuth, "no API key given").
					with(`Paste the key at the prompt, or pipe it in: echo "$KEY" | peppol-sh login`)
			}
			cred := &credential{key: key}
			s, err := a.connect(cred, true)
			if err != nil {
				return err
			}
			r, err := fetch[api.Account](cmd.Context(), s, getAccount)
			if err != nil {
				return err
			}
			if err := saveKey(a.env.ConfigDir, key); err != nil {
				return err
			}
			a.info("Logged in as %s (%s, key %s).", orDash(r.data.Email), cred.environment(), cred.masked())
			a.warnEnvKey()
			return nil
		},
	}
}

// readKey reads the key from a hidden prompt on a terminal, else from the
// first line of stdin.
func (a *app) readKey() (string, error) {
	if f, ok := a.env.Stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		fmt.Fprint(a.env.Stderr, "API key: ")
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(a.env.Stderr)
		return strings.TrimSpace(string(b)), err
	}
	line, err := bufio.NewReader(a.env.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (a *app) warnEnvKey() {
	if a.env.Getenv(envAPIKey) != "" {
		a.info("Note: %s is set, and it wins over the stored key.", envAPIKey)
	}
}

func newLogoutCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored API key",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			removed, err := removeKey(a.env.ConfigDir)
			if err != nil {
				return err
			}
			if removed {
				a.info("Removed the stored API key from %s.", configPath(a.env.ConfigDir))
			} else {
				a.info("No API key is stored.")
			}
			a.warnEnvKey()
			return nil
		},
	}
}
