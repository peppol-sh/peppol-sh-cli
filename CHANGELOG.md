# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - Unreleased

First version. Date this heading to release it; the mirror workflow then tags
`v0.1.0`.

### Added

- Commands: `version`, `health`, `login`, `logout`, `whoami`, `companies list`,
  `lookup` and `completion`.
- The API key comes from `--api-key`, then `PEPPOL_SH_API_KEY`, then the
  config file that `login` writes (`peppol-sh/config.toml` in the config
  directory of the OS, mode `0600`). A key is never printed in full.
- The key prefix selects the host: `ps_test_` → `https://sandbox.peppol.sh`,
  `ps_live_` → `https://api.peppol.sh`. `--base-url` and `PEPPOL_SH_BASE_URL`
  override it. `health` and `lookup` ignore a key with another prefix, with a
  warning.
- `--json` prints the API response body unchanged. Tables go to stdout,
  messages and errors to stderr; no colour when stdout is not a terminal or
  `NO_COLOR` is set.
- Exit codes: `0` success, `1` API or runtime error, `2` usage error, `3`
  authentication error.
- On HTTP 429 the CLI waits `Retry-After` (60 seconds at most) and tries one
  more time.
- Install channels: the Homebrew tap (`brew install peppol-sh/tap/peppol-sh`),
  the install scripts `install.sh` and `install.ps1`, and the archives on
  GitHub Releases with `checksums.txt`, SBOMs and artifact attestations. The
  macOS binaries are not signed or notarized; the cask removes the quarantine
  attribute.
