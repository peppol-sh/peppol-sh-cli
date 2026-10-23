# AGENTS.md

Guidance for AI coding agents that use or change `peppol-sh`, the CLI for the
[peppol.sh](https://peppol.sh) Peppol e-invoicing API. `README.md` has the
full reference; this file has the rules that prevent the common mistakes.

## Use the CLI

```bash
brew install peppol-sh/tap/peppol-sh
# or: curl -fsSL https://raw.githubusercontent.com/peppol-sh/peppol-sh-cli/main/install.sh | sh
```

```bash
export PEPPOL_SH_API_KEY=ps_test_...   # from a secret, never from a source file
peppol-sh whoami --json
```

- **Auth is an API key.** The key comes from `--api-key`, then
  `PEPPOL_SH_API_KEY`, then the config file of `peppol-sh login`; the first
  one that is set wins. In CI or an agent session, set `PEPPOL_SH_API_KEY`;
  `login` is for a person at a terminal.
- **The key prefix selects the host.** `ps_test_` goes to
  `https://sandbox.peppol.sh` (no real Peppol network), `ps_live_` to
  `https://api.peppol.sh`. Use a `ps_test_` key for every test. `--base-url`
  or `PEPPOL_SH_BASE_URL` overrides the host (staging
  `https://staging-api.peppol.sh`, local `http://localhost:8787`).
- **Read results with `--json`.** It prints the API response body on stdout,
  unchanged; the keys are the `snake_case` names of the JSON API. The tables
  without `--json` are for people and can change. Messages and errors go to
  stderr.
- **Branch on the exit code**, not on the message text: `0` success, `1` API,
  runtime or network error, `2` usage error, `3` authentication error (no
  key or an unknown key prefix for a command that needs a key, HTTP 401 or
  403).
- **Do not add a retry loop for 429.** The CLI waits `Retry-After` (60 s at
  most) and tries once more itself. A longer `Retry-After` fails at once with
  the wait in the message.
- **Do not invent commands.** The complete surface is below. To send a
  document or manage webhooks, use the API or an SDK.

| Command | API call (`operationId`) | Needs a key |
|---|---|---|
| `version` (also `--version`) | none | no |
| `health` | `GET /v1/health` (`getHealth`) | no |
| `login` | `GET /v1/account` (`getAccount`), then stores the key | the key to store |
| `logout` | none; removes the stored key | no |
| `whoami` | `GET /v1/account` (`getAccount`) | yes |
| `companies list` | `GET /v1/companies` (`listCompanies`) | yes |
| `lookup <peppol_id> [--domain sml\|smk]` | `GET /v1/lookup/{peppol_id}` (`lookupParticipant`) | no; sent when set, for a higher rate limit |
| `completion bash\|zsh\|fish\|powershell` | none | no |

## Change the CLI

This repository is a read-only mirror: CI force-pushes the `packages/cli`
folder of the private peppol.sh monorepo over it, so commit hashes change and
a commit made here is lost. Open an issue or a pull request here; a maintainer
applies it upstream.

```bash
gofmt -l .          # prints nothing
go vet ./...
go test ./...
goreleaser check    # .goreleaser.yaml is valid
```

### Layout

| Path | Contents |
|---|---|
| `main.go` | Entry point. Calls the `cli` package and exits with its code |
| `internal/cli/` | The cobra commands, key and host resolution, output, error rendering, exit codes. `cli.Run` is the test seam |
| `internal/api/client.gen.go` | The generated API client. Do not edit |
| `internal/api/generate.go`, `internal/api/oapi-codegen.yaml` | The `go:generate` line and the generator config |
| `internal/spectest/` | Finds `openapi.yaml` for the drift and contract tests |
| `internal/version/version.go` | `var Version`, the only place where the code knows its version. The release sets it with `-ldflags -X` |
| `install.sh`, `install.ps1` | The install scripts |
| `.goreleaser.yaml`, `.github/workflows/` | Build, release and the CI of the mirror |

### Add a command

1. **Regenerate the client** if the command needs an operation that it does
   not have. In the monorepo: `cd packages/cli && go generate ./...`. It reads
   `apps/api/src/openapi.yaml` and writes `internal/api/client.gen.go`. Commit
   the output.
2. **Add the command** in `internal/cli`, with the `operationId` annotation of
   the operation that it calls. Send the request with `request` (or `fetch`)
   in `internal/cli/request.go`. A command that calls no API goes in the
   explicit list of commands with no API call.
3. **Add its row to the contract test table.** The contract test runs the
   command against a recording transport and checks that the method and the
   path agree with the operation in `openapi.yaml`.
4. **Test it through `cli.Run`**, against an `httptest.Server` (with
   `--base-url`) or a recording `http.RoundTripper`. Inject the sleep, the
   environment and the config directory. Expected values come from the spec or
   a worked example, not from the code under test.
5. **Add it to `CHANGELOG.md`** under `## [Unreleased]`, and to the command
   tables in this file and in `README.md`.

### Rules

- **The behaviour is shared.** A new command gets key resolution, host
  selection, `--json`, error rendering, exit codes, the 429 retry and the
  `User-Agent` (`peppol-sh/<version> (<GOOS>/<GOARCH>)`) from `internal/cli`.
  Use them; do not write a second version.
- **`--json` is the raw body.** Do not decode and re-encode it.
- **Never print a full key.** Mask it as the prefix, `...` and the last four
  characters (`ps_test_...a1b2`).
- **Use the words of the product:** account, workspace, company, document,
  sandbox, live. A message never says "tenant" or "provider".
- **Two tests skip in the mirror.** The drift test (the committed
  `client.gen.go` equals the generator output) and the contract test need
  `../../apps/api/src/openapi.yaml`, which exists only in the monorepo. They
  skip without it. With `PEPPOL_SH_REQUIRE_DRIFT_CHECK=1` (monorepo CI sets
  it) a missing spec is a failure. A skip in the mirror is correct; a skip in
  the monorepo is a defect. A new test that needs the spec gets its path from
  `spectest.Path`.
- **No test calls the network.**
- **Self-contained.** The module is `github.com/peppol-sh/peppol-sh-cli`. It
  imports nothing from the monorepo.
- **No pagination code yet.** `GET /v1/companies` returns every company in
  one `data` array. Cursor pagination comes with the first list command whose
  endpoint has a cursor.

### Release

Set `var Version = "X.Y.Z"` in `internal/version/version.go`, add the dated
heading `## [X.Y.Z] - YYYY-MM-DD` to `CHANGELOG.md` (the version test checks
that the two agree), and merge to `main` in the monorepo. The mirror then gets
the tag `vX.Y.Z`, which starts `.github/workflows/release.yml`. Never move a
tag. The full procedure, with the one-time setup and failed releases, is
`docs/runbooks/cli-release.md` in the monorepo.

### Not in v0, on purpose

Do not add these without a decision from a maintainer:

- No OS keychain. The key is in the config file (mode `0600`).
- No profiles and no second stored key.
- No browser login. `login` takes an API key.
- No telemetry and no self-update. Updates come from the package manager.
- No other command groups (documents, webhooks, workspaces, …) in v0.
