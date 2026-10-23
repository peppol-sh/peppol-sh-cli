#!/bin/sh
# Installs the peppol.sh CLI (peppol-sh) on Linux and macOS.
#
#   curl -fsSL https://raw.githubusercontent.com/peppol-sh/peppol-sh-cli/main/install.sh | sh
#
# Environment (all optional):
#   PEPPOL_SH_VERSION        version to install, e.g. 0.1.0 (default: the latest release)
#   PEPPOL_SH_INSTALL_DIR    target directory (default: /usr/local/bin when
#                            writable, else $HOME/.local/bin)
#   PEPPOL_SH_DOWNLOAD_BASE  for tests and mirrors: URL of a directory with the
#                            archives and checksums.txt (default: the GitHub
#                            release). Both files come from there, so use only
#                            a trusted https URL.
#
# The archive is checked against the sha256 in checksums.txt before install.

set -eu

REPO="peppol-sh/peppol-sh-cli"
BINARY="peppol-sh"

say() {
	printf '%s\n' "$*" >&2
}

fail() {
	say "install.sh: error: $*"
	exit 1
}

has() {
	command -v "$1" >/dev/null 2>&1
}

detect_os() {
	case "$(uname -s)" in
	Linux) echo linux ;;
	Darwin) echo darwin ;;
	*) fail "unsupported OS $(uname -s): use install.ps1 on Windows, or download an archive from https://github.com/$REPO/releases" ;;
	esac
}

detect_arch() {
	arch="$(uname -m)"
	# A shell under Rosetta 2 reports x86_64 on Apple silicon.
	if [ "$arch" = x86_64 ] && [ "$(uname -s)" = Darwin ] &&
		[ "$(sysctl -n sysctl.proc_translated 2>/dev/null || echo 0)" = 1 ]; then
		arch=arm64
	fi
	case "$arch" in
	x86_64 | amd64) echo amd64 ;;
	arm64 | aarch64) echo arm64 ;;
	*) fail "unsupported architecture $arch (supported: amd64, arm64)" ;;
	esac
}

download() {
	# $1 URL, $2 output file
	if has curl; then
		case "$1" in
		https://*) curl --proto '=https' --tlsv1.2 -fsSL -o "$2" "$1" ;;
		*) curl -fsSL -o "$2" "$1" ;;
		esac
	elif has wget; then
		wget -q -O "$2" "$1"
	else
		fail "curl or wget is required"
	fi
}

sha256_of() {
	if has sha256sum; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif has shasum; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		fail "sha256sum or shasum is required to verify the download"
	fi
}

default_install_dir() {
	if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
		echo /usr/local/bin
	else
		echo "${HOME:?HOME is not set}/.local/bin"
	fi
}

main() {
	os="$(detect_os)"
	arch="$(detect_arch)"
	archive="${BINARY}_${os}_${arch}.tar.gz"

	version="${PEPPOL_SH_VERSION:-}"
	version="${version#v}"
	if [ -n "${PEPPOL_SH_DOWNLOAD_BASE:-}" ]; then
		base="${PEPPOL_SH_DOWNLOAD_BASE%/}"
	elif [ -n "$version" ]; then
		base="https://github.com/$REPO/releases/download/v$version"
	else
		base="https://github.com/$REPO/releases/latest/download"
	fi

	install_dir="${PEPPOL_SH_INSTALL_DIR:-$(default_install_dir)}"

	tmp="$(mktemp -d 2>/dev/null || mktemp -d -t peppol-sh)"
	trap 'rm -rf "$tmp"' EXIT
	trap 'exit 130' INT
	trap 'exit 143' TERM

	say "Downloading $base/$archive"
	download "$base/$archive" "$tmp/$archive" || fail "cannot download $base/$archive"
	download "$base/checksums.txt" "$tmp/checksums.txt" || fail "cannot download $base/checksums.txt"

	expected="$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")"
	[ -n "$expected" ] || fail "checksums.txt has no entry for $archive"
	actual="$(sha256_of "$tmp/$archive")"
	if [ "$actual" != "$expected" ]; then
		fail "checksum mismatch for $archive (expected $expected, got $actual)"
	fi

	tar -xzf "$tmp/$archive" -C "$tmp"
	[ -f "$tmp/$BINARY" ] || fail "$archive does not contain $BINARY"

	mkdir -p "$install_dir"
	install -m 0755 "$tmp/$BINARY" "$install_dir/$BINARY"
	say "Installed $install_dir/$BINARY"

	case ":${PATH:-}:" in
	*":$install_dir:"*) ;;
	*) say "Note: $install_dir is not on your PATH. Add it, e.g.: export PATH=\"$install_dir:\$PATH\"" ;;
	esac

	"$install_dir/$BINARY" version
}

# Everything runs from main, so a download cut short by `curl | sh` runs nothing.
main "$@"
