// Command peppol-sh is the command-line client of the peppol.sh API.
package main

import (
	"os"

	"github.com/peppol-sh/peppol-sh-cli/internal/cli"
)

func main() {
	os.Exit(cli.Main())
}
