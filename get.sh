#!/bin/sh
# Quick install: downloads the latest release and runs its install.sh.
#
#   curl -fsSL https://raw.githubusercontent.com/PericlesSavio/8bitdo-keymap/main/get.sh | sh
#
# Works on Debian, Fedora, Arch, openSUSE and their derivatives (x86_64 and
# aarch64). No Go toolchain needed: release archives ship prebuilt binaries.
set -e

REPO=PericlesSavio/8bitdo-keymap

case "$(uname -m)" in
x86_64 | amd64) ARCH=amd64 ;;
aarch64 | arm64) ARCH=arm64 ;;
*)
	echo "error: unsupported architecture $(uname -m)" >&2
	exit 1
	;;
esac

if command -v curl >/dev/null; then
	fetch() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null; then
	fetch() { wget -qO "$2" "$1"; }
else
	echo "error: curl or wget is required" >&2
	exit 1
fi

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

URL="https://github.com/$REPO/releases/latest/download/8bitdo-keymap-linux-$ARCH.tar.gz"
echo "==> Downloading $URL"
fetch "$URL" "$TMP/release.tar.gz"
tar -xzf "$TMP/release.tar.gz" -C "$TMP"

# sudo inside install.sh asks for the password on the terminal, not on the pipe.
sh "$TMP/8bitdo-keymap/install.sh" </dev/tty
