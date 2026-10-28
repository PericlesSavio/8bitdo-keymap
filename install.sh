#!/bin/sh
# Installs 8bitdoctl: daemon, CLI, Qt GUI and application menu integration.
#
#   ./install.sh              install everything
#   ./install.sh --uninstall  remove everything
#
# Run as your regular user; sudo is used only for the steps that need it. The
# daemon itself runs as your user (systemd user service), not as root.
set -e

cd "$(dirname "$0")"

if [ "$(id -u)" = 0 ]; then
	echo "error: run as your regular user (the daemon is a per-user service)" >&2
	exit 1
fi

if [ "$1" = "--uninstall" ]; then
	exec sh packaging/8bitdoctl-uninstall
fi

APPDIR=/usr/share/8bitdoctl
MODULE=github.com/PericlesSavio/8bitdoctl

if ! command -v sudo >/dev/null; then
	echo "error: sudo is required (install it and add your user to the sudo/wheel group)" >&2
	exit 1
fi

# Release archives ship prebuilt binaries; a source checkout is built here.
if [ -x 8bitdod ] && [ -x 8bitdoctl ]; then
	echo "==> Using prebuilt binaries"
else
	echo "==> Checking dependencies"
	if ! command -v go >/dev/null; then
		echo "error: Go >= 1.21 is required (https://go.dev/dl or your distro's 'go'/'golang' package)" >&2
		exit 1
	fi

	echo "==> Building"
	VERSION=0.5.0
	CGO_ENABLED=0 go build -ldflags "-X '$MODULE/internal/version.Version=$VERSION'" ./cmd/8bitdod
	CGO_ENABLED=0 go build -ldflags "-X '$MODULE/internal/version.Version=$VERSION'" ./cmd/8bitdoctl
fi
./8bitdoctl completion bash >8bitdoctl.bash

echo "==> Installing daemon, CLI and system files (root needed)"
sudo install -m 755 8bitdod 8bitdoctl /usr/bin
sudo install -Dm644 8bitdoctl.bash /etc/bash_completion.d/8bitdoctl
rm -f 8bitdod 8bitdoctl 8bitdoctl.bash
sudo install -Dm644 etc/8bitdod.service /usr/lib/systemd/user/8bitdod.service
sudo install -Dm644 etc/io.github.PericlesSavio.EightBitDoCtl.service \
	/usr/share/dbus-1/services/io.github.PericlesSavio.EightBitDoCtl.service
sudo install -Dm644 etc/60-8bitdoctl.rules /usr/lib/udev/rules.d/60-8bitdoctl.rules
sudo udevadm control --reload
sudo udevadm trigger --action=change --subsystem-match=misc --sysname-match=uinput

echo "==> Installing the Qt GUI and menu entry"
sudo install -Dm644 gui-qt/main.py "$APPDIR/gui-qt/main.py"
sudo install -Dm755 packaging/8bitdoctl-qt /usr/bin/8bitdoctl-qt
sudo install -Dm755 packaging/8bitdoctl-uninstall /usr/bin/8bitdoctl-uninstall
sudo install -Dm644 packaging/8bitdoctl-qt.desktop /usr/share/applications/8bitdoctl-qt.desktop
sudo update-desktop-database /usr/share/applications 2>/dev/null || true

if ! python3 -c "import PySide6" 2>/dev/null; then
	echo "==> Installing PySide6 for the GUI"
	# ID_LIKE makes derivatives (Mint, Pop!_OS, Manjaro, Rocky...) resolve to their base.
	. /etc/os-release
	case " $ID $ID_LIKE " in
	*" debian "* | *" ubuntu "*) sudo apt-get install -y python3-pyside6.qtwidgets || true ;;
	*" fedora "* | *" rhel "*) sudo dnf install -y python3-pyside6 || true ;;
	*" arch "*) sudo pacman -S --needed --noconfirm pyside6 || true ;;
	*" suse "* | *" opensuse "*) sudo zypper --non-interactive install python3-PySide6 || true ;;
	esac
	# Fallback for distros without a PySide6 package (e.g. Ubuntu 24.04 and its
	# derivatives): a private venv that the 8bitdoctl-qt launcher prefers.
	if ! python3 -c "import PySide6" 2>/dev/null; then
		echo "==> No PySide6 package found; installing it into $APPDIR/venv"
		if ! python3 -c "import venv, ensurepip" 2>/dev/null && command -v apt-get >/dev/null; then
			sudo apt-get install -y python3-venv || true
		fi
		if ! { sudo python3 -m venv "$APPDIR/venv" &&
			sudo "$APPDIR/venv/bin/pip" install --quiet PySide6-Essentials; }; then
			sudo rm -rf "$APPDIR/venv"
			echo "warning: could not install PySide6; the CLI works, but the GUI needs it." >&2
		fi
	fi
fi

echo "==> Enabling the daemon for your user"
systemctl --user daemon-reload
systemctl --user enable 8bitdod.service
systemctl --user restart 8bitdod.service

echo
echo "Done! You can now:"
echo "  - run '8bitdoctl mapping list' in the terminal"
echo "  - open \"8BitDo Key Mapper\" in the application menu"
