# peppol-sh

The command-line tool for the [peppol.sh](https://peppol.sh) Peppol
e-invoicing API. Check your key, list your companies and look up a Peppol
participant from a terminal, a CI job or an AI coding agent.

[![Release](https://img.shields.io/github/v/release/peppol-sh/peppol-sh-cli)](https://github.com/peppol-sh/peppol-sh-cli/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)

`peppol-sh` is one static binary for Linux, macOS and Windows (amd64 and
arm64). It talks to the `/v1` API over HTTPS. Its API client is generated
from the same OpenAPI contract as the API.

## Install

### Homebrew (macOS, Linux)

```bash
brew install peppol-sh/tap/peppol-sh
```

The macOS binaries are signed with an Apple Developer ID and notarized.
Homebrew also installs the shell completion for bash, zsh and fish.

### Install script (Linux, macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/peppol-sh/peppol-sh-cli/main/install.sh | sh
```

### Install script (Windows, PowerShell)

```powershell
irm https://raw.githubusercontent.com/peppol-sh/peppol-sh-cli/main/install.ps1 | iex
```

Both scripts download the archive for your OS and CPU from the latest GitHub
Release and check its SHA-256 against `checksums.txt` before they install the
binary. `install.ps1` also adds its directory to your user `PATH`. Environment
variables change what they do:

| Variable | Default | What it does |
| --- | --- | --- |
| `PEPPOL_SH_VERSION` | the latest release | Install this version, for example `0.1.0`. |
| `PEPPOL_SH_INSTALL_DIR` | `install.sh`: `/usr/local/bin` if you can write to it, else `~/.local/bin`. `install.ps1`: `%LOCALAPPDATA%\peppol-sh\bin` | Put the binary in this directory. |
| `PEPPOL_SH_DOWNLOAD_BASE` | the GitHub Release | For tests and mirrors: download the archive and `checksums.txt` from this URL. Both come from there, so use only a trusted `https` URL. |

```bash
curl -fsSL https://raw.githubusercontent.com/peppol-sh/peppol-sh-cli/main/install.sh \
  | PEPPOL_SH_VERSION=0.1.0 PEPPOL_SH_INSTALL_DIR="$HOME/bin" sh
```

### Manual download

Download the archive for your platform from
[GitHub Releases](https://github.com/peppol-sh/peppol-sh-cli/releases/latest):
`peppol-sh_<os>_<arch>.tar.gz` (`peppol-sh_windows_<arch>.zip` on Windows),
with `<os>` one of `linux`, `darwin`, `windows` and `<arch>` one of `amd64`,
`arm64`. Check it, unpack it, and put `peppol-sh` on your `PATH`.

```bash
sha256sum --ignore-missing -c checksums.txt          # macOS: shasum -a 256 --ignore-missing -c checksums.txt
gh attestation verify peppol-sh_linux_amd64.tar.gz --repo peppol-sh/peppol-sh-cli
```

Each archive has an SBOM and a GitHub artifact attestation, so
`gh attestation verify` proves that the release workflow of this repo built
it.

### Updates

`peppol-sh` does not update itself. Use the channel that installed it:
`brew upgrade peppol-sh`, or run the install script again.

## Quick start

You need an API key. Get a sandbox key (`ps_test_…`) from the dashboard at
<https://app.peppol.sh>, or with one API call:

```bash
curl -X POST https://api.peppol.sh/v1/signup \
  -H "Content-Type: application/json" \
  -d '{"email": "you@example.com"}'
# { "id": "acc_…", "api_key": "ps_test_…", … }  (the key is shown one time only)
```

Then:

```bash
peppol-sh login                      # asks for the key, checks it, stores it
peppol-sh whoami                     # the account of the key
peppol-sh companies list             # the companies in your workspace
peppol-sh lookup 0208:0123456749     # a participant on the Peppol network

peppol-sh companies list --json | jq -r '.data[].id'
```

## Authentication

Every command that calls the API sends the key as
`Authorization: Bearer <key>`. The key comes from the first of these that is
set:

1. the `--api-key` flag,
2. the `PEPPOL_SH_API_KEY` environment variable,
3. the config file that `peppol-sh login` wrote.

`peppol-sh login` checks the key with `GET /v1/account` before it stores it.
`peppol-sh logout` removes the stored key. In CI, do not log in: set
`PEPPOL_SH_API_KEY` from a secret. Other processes on the machine can see the
value of `--api-key`, so `PEPPOL_SH_API_KEY` or `login` is safer.

`peppol-sh` never prints a full key. It shows the prefix and the last four
characters, for example `ps_test_...a1b2`.

`health`, `lookup`, `version` and `completion` work with no key. `lookup`
sends a key when it finds one, because a key gives a higher rate limit.

## Sandbox and live

The key prefix selects the host:

| Key prefix | Host | Delivery |
| --- | --- | --- |
| `ps_test_` | `https://sandbox.peppol.sh` | Sandbox: never touches the real Peppol network |
| `ps_live_` | `https://api.peppol.sh` | Live: real Peppol delivery |
| no key | `https://api.peppol.sh` | Only the commands that need no key |

A key with any other prefix is an authentication error (exit code 3).
`health` and `lookup` need no key, so they ignore it with a warning.

To use another host, set `--base-url` or `PEPPOL_SH_BASE_URL` (the flag wins).
For example, staging is `https://staging-api.peppol.sh` and a local API is
`http://localhost:8787`:

```bash
PEPPOL_SH_BASE_URL=http://localhost:8787 peppol-sh health
```

## Config file

`peppol-sh login` writes the key to `peppol-sh/config.toml` in the config
directory of your OS:

| OS | Path |
| --- | --- |
| Linux | `$XDG_CONFIG_HOME/peppol-sh/config.toml` (default `~/.config/peppol-sh/config.toml`) |
| macOS | `~/Library/Application Support/peppol-sh/config.toml` |
| Windows | `%APPDATA%\peppol-sh\config.toml` |

The file has one key, `api_key`. It is readable only by you (mode `0600`, its
directory `0700`). `peppol-sh` does not use the OS keychain.

## Output

- By default, results are tables on stdout.
- `--json` prints the API response body to stdout as it came, for `jq` or a
  script. `peppol-sh version --json` prints the version, the OS and the CPU.
- Messages, hints and errors go to stderr, so stdout stays clean in a pipe.
- There is no colour when stdout is not a terminal or `NO_COLOR` is set.

## Errors and exit codes

An API error shows its `error.code`, its `error.message` and the HTTP status.
On a 5xx it also shows the `request_id`, when the API sent one; quote it when
you contact support.

| Exit code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | API or runtime error, also a network error |
| `2` | Usage error: an unknown command, a bad argument or a bad flag |
| `3` | Authentication error: no key or a key with an unknown prefix for a command that needs a key, or HTTP 401 or 403 |

On HTTP 429, `peppol-sh` waits the `Retry-After` seconds and tries one more
time. When the wait is longer than 60 seconds, or the second try gets 429
too, it stops with a message that says how long to wait.

## Commands

| Command | API call | Needs a key |
| --- | --- | --- |
| `peppol-sh version` (also `peppol-sh --version`) | none | no |
| `peppol-sh health` | `GET /v1/health` | no |
| `peppol-sh login` | `GET /v1/account`, then stores the key | the key to store |
| `peppol-sh logout` | none; removes the stored key | no |
| `peppol-sh whoami` | `GET /v1/account` | yes |
| `peppol-sh companies list` | `GET /v1/companies` | yes |
| `peppol-sh lookup <peppol_id> [--domain sml\|smk]` | `GET /v1/lookup/{peppol_id}` | no |
| `peppol-sh completion bash\|zsh\|fish\|powershell` | none | no |

`lookup` takes a Peppol ID in `scheme:id` form. `--domain sml` (the default)
is the production Peppol network; `--domain smk` is the Peppol test network.

Every command takes these flags:

| Flag | What it does |
| --- | --- |
| `--api-key <key>` | Use this key (see [Authentication](#authentication)). |
| `--base-url <url>` | Use this host (see [Sandbox and live](#sandbox-and-live)). |
| `--json` | Print JSON on stdout (see [Output](#output)). |

Run `peppol-sh <command> --help` for the flags of a command.

## Shell completion

Homebrew installs the completion for bash, zsh and fish. After another
install, write the script yourself:

```bash
# bash (needs the bash-completion package)
peppol-sh completion bash | sudo tee /etc/bash_completion.d/peppol-sh >/dev/null

# zsh
peppol-sh completion zsh > "${fpath[1]}/_peppol-sh"

# fish
peppol-sh completion fish > ~/.config/fish/completions/peppol-sh.fish
```

```powershell
# PowerShell: add this line to your $PROFILE
peppol-sh completion powershell | Out-String | Invoke-Expression
```

Open a new shell after you install a completion script.

## Privacy

`peppol-sh` has no telemetry. It sends a request only to the host above, and
only for the command you run. Each request has the header
`User-Agent: peppol-sh/<version> (<os>/<arch>)`.

## Documentation

- Guides and API reference — <https://peppol.sh/docs>
- OpenAPI spec — <https://api.peppol.sh/v1/openapi.json>
- Changelog — [CHANGELOG.md](./CHANGELOG.md)

## Contributing

This repository is a read-only mirror of a private monorepo, where
`peppol-sh` is developed next to the API and its OpenAPI contract. Issues and
pull requests are welcome here: a maintainer applies an accepted change
upstream, and it comes back with the next mirror push under a new commit hash.
Mirror pushes rewrite history, so do not keep long-lived branches here.

## License

MIT © e-invoice bv
